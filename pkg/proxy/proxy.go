package proxy

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/goproxyio/goproxy/pkg/modfetch"
	"github.com/goproxyio/goproxy/pkg/modfetch/codehost"
	"github.com/goproxyio/goproxy/pkg/module"
)

var cacheDir string
var innerHandle http.Handler

func NewProxy(cache string) http.Handler {
	modfetch.PkgMod = filepath.Join(cache, "pkg", "mod")
	codehost.WorkRoot = filepath.Join(modfetch.PkgMod, "cache", "vcs")

	cacheDir = filepath.Join(modfetch.PkgMod, "cache", "download")
	innerHandle = http.FileServer(http.Dir(cacheDir))

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
					ReturnInternalServerError(w, err)
					return
				}
				modPath := strings.TrimPrefix(mod[0], "/")
				modPath, err := module.DecodePath(modPath)
				if err != nil {
					ReturnInternalServerError(w, err)
					return
				}
				// ignore the error, incorrect tag may be given
				// forward to inner.ServeHTTP
				if err := downloadMod(modPath, version); err != nil {
					errLogger.Printf("download get err %s", err)
				}
			}

			// fetch latest version
			if strings.HasSuffix(r.URL.Path, "/@latest") {
				modPath := strings.TrimSuffix(r.URL.Path, "/@latest")
				modPath = strings.TrimPrefix(modPath, "/")
				modPath, err := module.DecodePath(modPath)
				if err != nil {
					ReturnInternalServerError(w, err)
					return
				}
				repo, err := modfetch.Lookup(modPath)
				if err != nil {
					errLogger.Printf("lookup failed: %v", err)
					ReturnInternalServerError(w, err)
					return
				}
				rev, err := repo.Stat("latest")
				if err != nil {
					errLogger.Printf("latest failed: %v", err)
					return
				}
				if err := downloadMod(modPath, rev.Version); err != nil {
					errLogger.Printf("download get err %s", err)
				}

			}

			if strings.HasSuffix(r.URL.Path, "/@v/list") {
				// TODO
				_, _ = w.Write([]byte(""))
				return
			}
		}
		innerHandle.ServeHTTP(w, r)
	})
}

func downloadMod(modPath, version string) error {
	if _, err := modfetch.InfoFile(modPath, version); err != nil {
		return err
	}
	if _, err := modfetch.GoModFile(modPath, version); err != nil {
		return err
	}
	if _, err := modfetch.GoModSum(modPath, version); err != nil {
		return err
	}
	mod := module.Version{Path: modPath, Version: version}
	if _, err := modfetch.DownloadZip(mod); err != nil {
		return err
	}
	if a, err := modfetch.Download(mod); err != nil {
		return err
	} else {
		log.Printf("goproxy: download %s@%s to dir %s\n", modPath, version, a)
	}
	return nil
}
