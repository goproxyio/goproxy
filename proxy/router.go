package proxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
)

// A RouterOps provides the proxy host and the external pattern
type RouterOps struct {
	Pattern string
	Proxy   string
}

// A Router is the proxy HTTP server,
// which implements Route Filter to
// routing private module or public module .
type Router struct {
	srv   *Server
	proxy *httputil.ReverseProxy
	regex *regexp.Regexp
}

// NewRouter returns a new Router using the given operations.
func NewRouter(srv *Server, ops *RouterOps) *Router {
	rt := &Router{
		srv: srv,
	}
	if ops != nil {
		if remote, err := url.Parse(ops.Proxy); err == nil {
			rt.proxy = httputil.NewSingleHostReverseProxy(remote)
			rt.proxy.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		}
		if regex, err := regexp.Compile(ops.Pattern); err == nil {
			rt.regex = regex
		}
	}
	return rt
}

func (rt *Router) Direct(path string) bool {
	if rt.regex == nil {
		return true
	}
	return rt.regex.Match([]byte(path))
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rt.proxy != nil && rt.Direct(r.URL.Path) {
		fmt.Fprintf(os.Stdout, "[direct] %s\n", r.URL)
		rt.srv.ServeHTTP(w, r)
		return
	}
	fmt.Fprintf(os.Stdout, "[proxy] %s\n", r.URL)
	rt.proxy.ServeHTTP(w, r)
	return
}
