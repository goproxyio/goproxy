package proxy

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/goproxyio/goproxy/internal/cfg"
	"github.com/goproxyio/goproxy/internal/modfetch"
	"github.com/goproxyio/goproxy/internal/modfetch/codehost"
	"github.com/goproxyio/goproxy/internal/modload"
	"github.com/goproxyio/goproxy/internal/module"
)

var cacheDir string
var innerHandle http.Handler

type modInfo struct {
	module.Version
	suf string
}

func setupEnv(basedir string) {
	modfetch.QuietLookup = true // just to hide modfetch/cache.go#127
	modfetch.PkgMod = filepath.Join(basedir, "pkg", "mod")
	codehost.WorkRoot = filepath.Join(modfetch.PkgMod, "cache", "vcs")
	cfg.CmdName = "mod download" // just to hide modfetch/fetch.go#L87
}

func NewProxy(cache string) http.Handler {
	setupEnv(cache)

	cacheDir = filepath.Join(modfetch.PkgMod, "cache", "download")
	innerHandle = http.FileServer(http.Dir(cacheDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("goproxy: %s request %s\n", r.RemoteAddr, r.URL.Path)
		info, err := parseModInfoFromUrl(r.URL.Path)
		if err != nil {
			innerHandle.ServeHTTP(w, r)
			return
		}
		switch suf := info.suf; suf {
		case ".info", ".mod", ".zip":
			{
				if _, err := os.Stat(filepath.Join(cacheDir, r.URL.Path)); err == nil {
					// cache files exist on disk
					innerHandle.ServeHTTP(w, r)
					return
				}
				realMod, err := getQuery(info.Version.Path, info.Version.Version)
				if err != nil {
					errLogger.Printf("goproxy: lookup %s@%s get err %s", info.Path, info.Version.Version, err)
					ReturnBadRequest(w, err)
					return
				}
				if realMod.Path != info.Version.Path {
					log.Printf("goproxy: mod %s@%s may have subpath, just return to make client recurse", info.Path, info.Version.Version)
					ReturnSuccess(w, nil)
					return
				}
				switch suf {
				case ".info":
					{
						if revInfo, err := modfetch.Stat(realMod.Path, realMod.Version); err != nil {
							// use Stat instead of InfoFile, because when query-version is master, no infoFile here, maybe bug of go
							// TODO(hxzhao527): check whether InfoFile have a bug?
							errLogger.Printf("goproxy: fetch info %s@%s get err %s", info.Path, info.Version.Version, err)
							ReturnBadRequest(w, err)
						} else {
							ReturnJsonData(w, revInfo)
						}
					}
				case ".mod":
					{
						if modFile, err := modfetch.GoModFile(realMod.Path, realMod.Version); err != nil {
							errLogger.Printf("goproxy: fetch modfile %s@%s get err %s", info.Path, info.Version.Version, err)
							ReturnBadRequest(w, err)
						} else {
							http.ServeFile(w, r, modFile)
						}
					}
				case ".zip":
					{
						mod := module.Version{Path: realMod.Path, Version: realMod.Version}
						if zipFile, err := modfetch.DownloadZip(mod); err != nil {
							errLogger.Printf("goproxy: download zip %s@%s get err %s", info.Path, info.Version.Version, err)
							ReturnBadRequest(w, err)
						} else {
							http.ServeFile(w, r, zipFile)
						}
					}
				}
				return
			}
		case "/@v/list", "/@latest":
			{
				repo, err := modfetch.Lookup(info.Path)
				if err != nil {
					errLogger.Printf("goproxy: lookup failed: %v", err)
					ReturnInternalServerError(w, err)
					return
				}
				switch suf {
				case "/@v/list":
					if info, err := repo.Versions(""); err != nil {
						ReturnInternalServerError(w, err)
					} else {
						data := strings.Join(info, "\n")
						ReturnSuccess(w, []byte(data))
					}
				case "/@latest":
					modLatestInfo, err := repo.Latest()
					if err != nil {
						ReturnInternalServerError(w, err)
						return
					}
					ReturnJsonData(w, modLatestInfo)
				}
				return
			}
		}
	})
}

func parseModInfoFromUrl(url string) (*modInfo, error) {

	var modPath, modVersion, suf string
	var err error
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
		suf = path.Ext(url)
		tmp := strings.Split(url, "/@v/")
		if len(tmp) != 2 {
			return nil, fmt.Errorf("bad module path:%s", url)
		}
		modPath = strings.Trim(tmp[0], "/")
		modVersion = strings.TrimSuffix(tmp[1], suf)

		modVersion, err = module.DecodeVersion(modVersion)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("bad module path:%s", url)
	}
	// decode path & version, next proxy and source need
	modPath, err = module.DecodePath(modPath)
	if err != nil {
		return nil, err
	}

	return &modInfo{module.Version{Path: modPath, Version: modVersion}, suf}, nil
}

// getQuery evaluates the given package path, version pair
// to determine the underlying module version being requested.
// If forceModulePath is set, getQuery must interpret path
// as a module path.
func getQuery(path, vers string) (module.Version, error) {

	// First choice is always to assume path is a module path.
	// If that works out, we're done.
	info, err := modload.Query(path, vers, modload.Allowed)
	if err == nil {
		return module.Version{Path: path, Version: info.Version}, nil
	}

	// Otherwise, try a package path.
	m, _, err := modload.QueryPackage(path, vers, modload.Allowed)
	return m, err
}
