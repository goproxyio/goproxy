package proxy

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

// A RouterOps provides the proxy host and the external pattern
type RouterOptions struct {
	Pattern string
	Proxy   string
}

// A Router is the proxy HTTP server,
// which implements Route Filter to
// routing private module or public module .
type Router struct {
	srv     *Server
	proxy   *httputil.ReverseProxy
	pattern string
}

// NewRouter returns a new Router using the given operations.
func NewRouter(srv *Server, opts *RouterOptions) *Router {
	rt := &Router{
		srv: srv,
	}
	if opts != nil {
		if remote, err := url.Parse(opts.Proxy); err == nil {
			proxy := httputil.NewSingleHostReverseProxy(remote)
			director := proxy.Director
			proxy.Director = func(r *http.Request) {
				director(r)
				r.Host = remote.Host
			}
			rt.proxy = proxy

			rt.proxy.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		}
		rt.pattern = opts.Pattern
	}
	return rt
}

func (rt *Router) Direct(path string) bool {
	if rt.pattern == "" {
		return true
	}
	return GlobsMatchPath(rt.pattern, path)
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rt.proxy != nil && rt.Direct(r.URL.Path) {
		log.Printf("------ --- %s [direct]\n", r.URL)
		rt.srv.ServeHTTP(w, r)
		return
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
