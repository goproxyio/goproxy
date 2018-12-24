package proxy

import (
	"fmt"
	"github.com/goproxyio/goproxy/pkg/modfetch"
	"github.com/goproxyio/goproxy/pkg/module"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func NewProxy(cache string) http.Handler {
	modfetch.PkgMod = path.Join(cache, "pkg", "mod")
	cacheDir := path.Join(modfetch.PkgMod, "cache", "download")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("goproxy: %s download %s\n", r.RemoteAddr, r.URL.Path)
		if _, err := os.Stat(filepath.Join(cacheDir, r.URL.Path)); err != nil {
			suffix := path.Ext(r.URL.Path)
			if suffix == ".info" || suffix == ".mod" || suffix == ".zip" {
				mod := strings.Split(r.URL.Path, "/@v/")
				if len(mod) != 2 {
					ReturnBadRequest(w, fmt.Errorf("bad module path:%s", r.URL.Path))
					return
				}
				version := strings.TrimSuffix(mod[1], suffix)
				version, err = module.DecodeVersion(version)
				if err != nil {
					ReturnServerError(w, err)
					return
				}
				modPath := strings.TrimPrefix(mod[0], "/")
				modPath, err := module.DecodePath(modPath)
				if err != nil {
					ReturnServerError(w, err)
					return
				}
				// ignore the error, incorrect tag may be given
				// forward to inner.ServeHTTP
				_, _ = modfetch.Download(module.Version{Path: modPath, Version: version})
			}

			// fetch latest version
			if strings.HasSuffix(r.URL.Path, "/@latest") {
				modPath := strings.TrimSuffix(r.URL.Path, "/@latest")
				modPath = strings.TrimPrefix(modPath, "/")
				modPath, err := module.DecodePath(modPath)
				if err != nil {
					ReturnServerError(w, err)
					return
				}
				_, _ = modfetch.Download(module.Version{Path: modPath, Version: "latest"})
			}

			if strings.HasSuffix(r.URL.Path, "/@v/list") {
				// TODO
				_, _ = w.Write([]byte(""))
				return
			}
		}
		http.FileServer(http.Dir(cacheDir)).ServeHTTP(w, r)
	})
}
