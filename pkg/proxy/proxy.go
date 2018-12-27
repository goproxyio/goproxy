package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/goproxyio/goproxy/pkg/modfetch"
	"github.com/goproxyio/goproxy/pkg/modfetch/codehost"
	"github.com/goproxyio/goproxy/pkg/module"
)

var cacheDir string
var innerHandle http.Handler

type modInfo struct {
	module.Version
	suf string
}

func NewProxy(cache string) http.Handler {
	modfetch.PkgMod = filepath.Join(cache, "pkg", "mod")
	codehost.WorkRoot = filepath.Join(modfetch.PkgMod, "cache", "vcs")

	cacheDir = filepath.Join(modfetch.PkgMod, "cache", "download")
	innerHandle = http.FileServer(http.Dir(cacheDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("goproxy: %s request %s\n", r.RemoteAddr, r.URL.Path)
		encodedPath, err := encodeString(r.URL.Path)
		if err != nil {
			ReturnInternalServerError(w, err)
		}
		// use encoded path, innerHandle need this
		r.URL.Path = encodedPath
		if _, err := os.Stat(filepath.Join(cacheDir, encodedPath)); err != nil {
			info, err := parseModInfoFromUrl(encodedPath)
			if err != nil {
				ReturnBadRequest(w, err)
				return
			}
			switch suf := info.suf; suf {
			case "":
				// ignore the error, incorrect tag may be given
				// forward to inner.ServeHTTP
				if err := downloadMod(info.Path, info.Version.Version); err != nil {
					errLogger.Printf("goproxy: download %s@%s get err %s", info.Path, info.Version.Version, err)
				}
			case "/@v/list", "/@latest":
				repo, err := modfetch.Lookup(info.Path)
				if err != nil {
					errLogger.Printf("goproxy: lookup failed: %v", err)
					ReturnInternalServerError(w, err)
					return
				}
				switch suf {
				case "/@v/list":
					info, err := repo.Versions("")
					if err != nil {
						ReturnInternalServerError(w, err)
						return
					}
					data := strings.Join(info, "\n")
					ReturnSuccess(w, []byte(data))
					return
				case "/@latest":
					info, err := repo.Latest()
					if err != nil {
						ReturnInternalServerError(w, err)
						return
					}
					data, err := json.Marshal(info)
					if err != nil {
						// ignore
						errLogger.Printf("goproxy:  marshal mod version info get error: %s", err)
					}
					ReturnSuccess(w, data)
					return
				}
			}
		}
		innerHandle.ServeHTTP(w, r)
	})
}

func parseModInfoFromUrl(url string) (*modInfo, error) {

	var modPath, modVersion, suf string
	switch {
	case strings.HasSuffix(url, "/@v/list"):
		// /golang.org/x/net/@v/list
		suf = "/@v/list"
		modVersion = ""
		modPath = strings.Trim(strings.TrimSuffix(url, suf), "/")
	case strings.HasSuffix(url, "/@latest"):
		// /golang.org/x/@latest
		suf = "/@latest"
		modVersion = "latest"
		modPath = strings.Trim(strings.TrimSuffix(url, suf), "/")
	case strings.HasSuffix(url, ".info"), strings.HasSuffix(url, ".mod"), strings.HasSuffix(url, ".zip"):
		// /golang.org/x/net/@v/v0.0.0-20181220203305-927f97764cc3.info
		// /golang.org/x/net/@v/v0.0.0-20181220203305-927f97764cc3.mod
		// /golang.org/x/net/@v/v0.0.0-20181220203305-927f97764cc3.zip
		ext := path.Ext(url)
		tmp := strings.Split(url, "/@v/")
		if len(tmp) != 2 {
			return nil, fmt.Errorf("bad module path:%s", url)
		}
		modVersion = strings.TrimSuffix(tmp[1], ext)
		modPath = strings.Trim(tmp[0], "/")
	default:
		return nil, fmt.Errorf("bad module path:%s", url)
	}
	// decode path, next proxy and source need
	var err error
	modPath, err = module.DecodePath(modPath)
	if err != nil {
		return nil, err
	}
	modVersion, err = module.DecodeVersion(modVersion)
	if err != nil {
		return nil, err
	}
	return &modInfo{module.Version{Path: modPath, Version: modVersion}, suf}, nil
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

// encodeString will transform url.
// copy from [module.go@encodeString](https://github.com/golang/go/blob/master/src/cmd/go/internal/module/module.go#L437)
func encodeString(s string) (encoding string, err error) {
	haveUpper := false
	for _, r := range s {
		if r == '!' || r >= utf8.RuneSelf {
			// This should be disallowed by CheckPath, but diagnose anyway.
			// The correctness of the encoding loop below depends on it.
			return "", fmt.Errorf("internal error: inconsistency in EncodePath")
		}
		if 'A' <= r && r <= 'Z' {
			haveUpper = true
		}
	}

	if !haveUpper {
		return s, nil
	}

	var buf []byte
	for _, r := range s {
		if 'A' <= r && r <= 'Z' {
			buf = append(buf, '!', byte(r+'a'-'A'))
		} else {
			buf = append(buf, byte(r))
		}
	}
	return string(buf), nil
}
