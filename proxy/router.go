package proxy

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/goproxyio/goproxy/renameio"
)

const ListExpire = 5 * time.Minute

// A RouterOps provides the proxy host and the external pattern
type RouterOptions struct {
	Pattern      string
	Proxy        string
	DownloadRoot string
}

// A Router is the proxy HTTP server,
// which implements Route Filter to
// routing private module or public module .
type Router struct {
	srv          *Server
	proxy        *httputil.ReverseProxy
	pattern      string
	downloadRoot string
}

// NewRouter returns a new Router using the given operations.
func NewRouter(srv *Server, opts *RouterOptions) *Router {
	rt := &Router{
		srv: srv,
	}
	if opts != nil {
		if opts.Proxy == "" {
			log.Printf("not set proxy, all direct.")
			return rt
		}
		remote, err := url.Parse(opts.Proxy)
		if err != nil {
			log.Printf("parse proxy fail, all direct.")
			return rt
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		director := proxy.Director
		proxy.Director = func(r *http.Request) {
			director(r)
			r.Host = remote.Host
		}
		rt.proxy = proxy

		rt.proxy.Transport = &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		rt.proxy.ModifyResponse = func(r *http.Response) error {
			if r.StatusCode == http.StatusOK {
				var buf []byte
				if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
					gr, err := gzip.NewReader(r.Body)
					if err != nil {
						return err
					}
					defer gr.Close()
					buf, err = ioutil.ReadAll(gr)
					if err != nil {
						return err
					}
					r.Header.Del("Content-Encoding")
				} else {
					buf, err = ioutil.ReadAll(r.Body)
					if err != nil {
						return err
					}
				}
				r.Body = ioutil.NopCloser(bytes.NewReader(buf))
				if buf != nil {
					file := filepath.Join(opts.DownloadRoot, r.Request.URL.Path)
					os.MkdirAll(path.Dir(file), os.ModePerm)
					err = renameio.WriteFile(file, buf, 0666)
					if err != nil {
						return err
					}
				}
			}
			return nil
		}
		rt.pattern = opts.Pattern
		rt.downloadRoot = opts.DownloadRoot
	}
	return rt
}

func (rt *Router) Direct(path string) bool {
	if rt.pattern == "" {
		return false
	}
	return GlobsMatchPath(rt.pattern, path)
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rt.proxy == nil || rt.Direct(strings.TrimPrefix(r.URL.Path, "/")) {
		log.Printf("------ --- %s [direct]\n", r.URL)
		rt.srv.ServeHTTP(w, r)
		return
	}

	file := filepath.Join(rt.downloadRoot, r.URL.Path)
	if info, err := os.Stat(file); err == nil {
		if f, err := os.Open(file); err == nil {
			var ctype string
			defer f.Close()
			if strings.HasSuffix(r.URL.Path, "/@latest") {
				if time.Since(info.ModTime()) >= ListExpire {
					log.Printf("------ --- %s [proxy]\n", r.URL)
					rt.proxy.ServeHTTP(w, r)
				} else {
					ctype = "text/plain; charset=UTF-8"
					w.Header().Set("Content-Type", ctype)
					log.Printf("------ --- %s [cached]\n", r.URL)
					http.ServeContent(w, r, "", info.ModTime(), f)
				}
				return
			}

			i := strings.Index(r.URL.Path, "/@v/")
			if i < 0 {
				http.Error(w, "no such path", http.StatusNotFound)
				return
			}

			what := r.URL.Path[i+len("/@v/"):]
			if what == "list" {
				if time.Since(info.ModTime()) >= ListExpire {
					log.Printf("------ --- %s [proxy]\n", r.URL)
					rt.proxy.ServeHTTP(w, r)
					return
				} else {
					ctype = "text/plain; charset=UTF-8"
				}
			} else {
				ext := path.Ext(what)
				switch ext {
				case ".info":
					ctype = "application/json"
				case ".mod":
					ctype = "text/plain; charset=UTF-8"
				case ".zip":
					ctype = "application/octet-stream"
				default:
					http.Error(w, "request not recognized", http.StatusNotFound)
					return
				}
			}
			w.Header().Set("Content-Type", ctype)
			log.Printf("------ --- %s [cached]\n", r.URL)
			http.ServeContent(w, r, "", info.ModTime(), f)
			return
		}
	}
	log.Printf("------ --- %s [proxy]\n", r.URL)
	rt.proxy.ServeHTTP(w, r)
	return
}

// GlobsMatchPath reports whether any path prefix of target
// matches one of the glob patterns (as defined by path.Match)
// in the comma-separated globs list.
// It ignores any empty or malformed patterns in the list.
func GlobsMatchPath(globs, target string) bool {
	for globs != "" {
		// Extract next non-empty glob in comma-separated list.
		var glob string
		if i := strings.Index(globs, ","); i >= 0 {
			glob, globs = globs[:i], globs[i+1:]
		} else {
			glob, globs = globs, ""
		}
		if glob == "" {
			continue
		}

		// A glob with N+1 path elements (N slashes) needs to be matched
		// against the first N+1 path elements of target,
		// which end just before the N+1'th slash.
		n := strings.Count(glob, "/")
		prefix := target
		// Walk target, counting slashes, truncating at the N+1'th slash.
		for i := 0; i < len(target); i++ {
			if target[i] == '/' {
				if n == 0 {
					prefix = target[:i]
					break
				}
				n--
			}
		}
		if n > 0 {
			// Not enough prefix elements.
			continue
		}
		matched, _ := path.Match(glob, prefix)
		if matched {
			return true
		}
	}
	return false
}
