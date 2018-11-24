package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/goproxyio/goproxy/module"
)

var cacheDir string
var listen string

func init() {
	flag.StringVar(&listen, "listen", "0.0.0.0:8081", "service listen address")
	flag.Parse()
}

func main() {
	gpEnv := os.Getenv("GOPATH")
	if gpEnv == "" {
		panic("can not find $GOPATH")
	}
	fmt.Fprintf(os.Stdout, "goproxy: %s inited.\n", time.Now().Format("2006-01-02 15:04:05"))
	gp := filepath.SplitList(gpEnv)
	cacheDir = filepath.Join(gp[0], "pkg", "mod", "cache", "download")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stdout, "goproxy: %s cache dir is not exist. %s\n", time.Now().Format("2006-01-02 15:04:05"), cacheDir)
		os.MkdirAll(cacheDir, 0755)
	}
	http.Handle("/", mainHandler(http.FileServer(http.Dir(cacheDir))))
	err := http.ListenAndServe(listen, nil)
	if nil != err {
		panic(err)
	}
}

func mainHandler(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(os.Stdout, "goproxy: %s %s download %s\n", r.RemoteAddr, time.Now().Format("2006-01-02 15:04:05"), r.URL.Path)
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
				path := strings.TrimPrefix(mod[0], "/")
				path, err := module.DecodePath(path)
				if err != nil {
					ReturnServerError(w, err)
					return
				}
				// ignore the error, incorrect tag may be given
				// forward to inner.ServeHTTP
				goGet(path, version, suffix, w, r)
			}

			// fetch latest version
			if strings.HasSuffix(r.URL.Path, "/@latest") {
				path := strings.TrimSuffix(r.URL.Path, "/@latest")
				path = strings.TrimPrefix(path, "/")
				path, err := module.DecodePath(path)
				if err != nil {
					ReturnServerError(w, err)
					return
				}
				goGet(path, "latest", "", w, r)
			}

			if strings.HasSuffix(r.URL.Path, "/@v/list") {
				w.Write([]byte(""))
				return
			}
		}
		inner.ServeHTTP(w, r)
	})
}

func goGet(path, version, suffix string, w http.ResponseWriter, r *http.Request) error {
	cmd := exec.Command("go", "get", "-d", path+"@"+version)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	bytesErr, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}

	_, err = ioutil.ReadAll(stdout)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "goproxy: download %s stderr:\n%s", path, string(bytesErr))
		return err
	}
	out := fmt.Sprintf("%s", bytesErr)

	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) != 4 {
			continue
		}
		if f[1] == "downloading" && f[2] == path && f[3] != version && suffix != "" {
			h := r.Host
			mod := strings.Split(r.URL.Path, "/@v/")
			p := fmt.Sprintf("%s/@v/%s%s", mod[0], f[3], suffix)
			scheme := "http:"
			if r.TLS != nil {
				scheme = "https:"
			}
			url := fmt.Sprintf("%s//%s/%s", scheme, h, p)
			http.Redirect(w, r, url, 302)
		}
	}
	return nil
}
