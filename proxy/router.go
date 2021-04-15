package proxy

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
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

	"github.com/goproxyio/goproxy/v2/renameio"
	"github.com/goproxyio/goproxy/v2/sumdb"

	"github.com/prometheus/client_golang/prometheus"
)

// ListExpire list data expire data duration.
const ListExpire = 5 * time.Minute

// RouterOptions provides the proxy host and the external pattern
type RouterOptions struct {
	Pattern      string
	Proxy        string
	DownloadRoot string
}

// A Router is the proxy HTTP server,
// which implements Route Filter to
// routing private module or public module .
type Router struct {
	opts         *RouterOptions
	srv          *Server
	proxy        *httputil.ReverseProxy
	pattern      string
	downloadRoot string
}

func (router *Router) customModResponse(r *http.Response) error {
	var err error
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
			// rewrite content-length header due to the decompressed data will be refilled in the body
			r.Header.Set("Content-Length", fmt.Sprint(len(buf)))
		} else {
			buf, err = ioutil.ReadAll(r.Body)
			if err != nil {
				return err
			}
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(buf))
		if buf != nil {
			file := filepath.Join(router.opts.DownloadRoot, r.Request.URL.Path)
			os.MkdirAll(path.Dir(file), os.ModePerm)
			err = renameio.WriteFile(file, buf, 0666)
			if err != nil {
				return err
			}
		}
	}
	// support 302 status code.
	if r.StatusCode == http.StatusFound {
		loc := r.Header.Get("Location")
		if loc == "" {
			return fmt.Errorf("%d response missing Location header", r.StatusCode)
		}

		// TODO: location is relative.
		_, err := url.Parse(loc)
		if err != nil {
			return fmt.Errorf("failed to parse Location header %q: %v", loc, err)
		}
		resp, err := http.Get(loc)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var buf []byte
		if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			defer gr.Close()
			buf, err = ioutil.ReadAll(gr)
			if err != nil {
				return err
			}
			resp.Header.Del("Content-Encoding")
		} else {
			buf, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
		}
		resp.Body = ioutil.NopCloser(bytes.NewReader(buf))
		if buf != nil {
			file := filepath.Join(router.opts.DownloadRoot, r.Request.URL.Path)
			os.MkdirAll(path.Dir(file), os.ModePerm)
			err = renameio.WriteFile(file, buf, 0666)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// NewRouter returns a new Router using the given operations.
func NewRouter(srv *Server, opts *RouterOptions) *Router {
	rt := &Router{
		opts: opts,
		srv:  srv,
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
		rt.proxy.ModifyResponse = rt.customModResponse
		rt.pattern = opts.Pattern
		rt.downloadRoot = opts.DownloadRoot
	}
	return rt
}

// Direct decides whether a path should directly access.
func (rt *Router) Direct(path string) bool {
	if rt.pattern == "" {
		return false
	}
	return GlobsMatchPath(rt.pattern, path)
}

// ServveHTTP implements http handler.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mw := NewMetricsResponseWriter(w)
	// sumdb handler
	if strings.HasPrefix(r.URL.Path, "/sumdb/") {
		sumdb.Handler(mw, r)
		totalRequest.With(prometheus.Labels{"mode": "sumdb", "status": mw.status()}).Inc()
		return
	}

	if rt.proxy == nil || rt.Direct(strings.TrimPrefix(r.URL.Path, "/")) {
		log.Printf("------ --- %s [direct]\n", r.URL)
		rt.srv.ServeHTTP(mw, r)
		totalRequest.With(prometheus.Labels{"mode": "direct", "status": mw.status()}).Inc()
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
					rt.proxy.ServeHTTP(mw, r)
					totalRequest.With(prometheus.Labels{"mode": "proxy", "status": mw.status()}).Inc()
				} else {
					ctype = "text/plain; charset=UTF-8"
					mw.Header().Set("Content-Type", ctype)
					log.Printf("------ --- %s [cached]\n", r.URL)
					http.ServeContent(mw, r, "", info.ModTime(), f)
					totalRequest.With(prometheus.Labels{"mode": "cached", "status": mw.status()}).Inc()
				}
				return
			}

			i := strings.Index(r.URL.Path, "/@v/")
			if i < 0 {
				http.Error(mw, "no such path", http.StatusNotFound)
				totalRequest.With(prometheus.Labels{"mode": "proxy", "status": mw.status()}).Inc()
				return
			}

			what := r.URL.Path[i+len("/@v/"):]
			if what == "list" {
				if time.Since(info.ModTime()) >= ListExpire {
					log.Printf("------ --- %s [proxy]\n", r.URL)
					rt.proxy.ServeHTTP(mw, r)
					totalRequest.With(prometheus.Labels{"mode": "proxy", "status": mw.status()}).Inc()
					return
				}
				ctype = "text/plain; charset=UTF-8"
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
					http.Error(mw, "request not recognized", http.StatusNotFound)
					totalRequest.With(prometheus.Labels{"mode": "proxy", "status": mw.status()}).Inc()
					return
				}
			}
			mw.Header().Set("Content-Type", ctype)
			log.Printf("------ --- %s [cached]\n", r.URL)
			http.ServeContent(mw, r, "", info.ModTime(), f)
			totalRequest.With(prometheus.Labels{"mode": "cached", "status": mw.status()}).Inc()
			return
		}
	}
	log.Printf("------ --- %s [proxy]\n", r.URL)
	rt.proxy.ServeHTTP(mw, r)
	totalRequest.With(prometheus.Labels{"mode": "proxy", "status": mw.status()}).Inc()
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
