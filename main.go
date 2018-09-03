package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/goproxyio/goproxy/module"
)

var cacheDir string
var listen string

func init() {
	flag.StringVar(&listen, "listen", "0.0.0.0:8081", "service listen address")
	flag.Parse()
}

func main() {
	gp := os.Getenv("GOPATH")
	if gp == "" {
		panic("can not find $GOPATH")
	}
	cacheDir = filepath.Join(gp, "pkg", "mod", "cache", "download")
	http.Handle("/", mainHandler(http.FileServer(http.Dir(cacheDir))))
	err := http.ListenAndServe(listen, nil)
	if nil != err {
		panic(err)
	}
}

func mainHandler(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(filepath.Join(cacheDir, r.URL.Path)); err != nil {
			var suffix string
			if strings.HasSuffix(r.URL.Path, ".info") || strings.HasSuffix(r.URL.Path, ".mod") {
				suffix = ".mod"
				if strings.HasSuffix(r.URL.Path, ".info") {
					suffix = ".info"
				}
				mod := strings.Split(r.URL.Path, "/@v/")
				if len(mod) != 2 {
					ReturnServerError(w, fmt.Errorf("bad module path:%s", r.URL.Path))
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
				err = goGet(path, version, suffix, w, r)
				if err != nil {
					ReturnServerError(w, err)
					return
				}
			}
			if strings.HasSuffix(r.URL.Path, "/@v/list") {
				w.WriteHeader(200)
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

	bytesOut, err := ioutil.ReadAll(stdout)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	out := fmt.Sprintf("%s", bytesErr)

	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) != 4 {
			continue
		}
		if f[1] == "downloading" && f[2] == path && f[3] != version {
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

	fmt.Fprintf(os.Stdout, "goproxy: download %s stdout: %s stderr: %s\n", path, string(bytesOut), string(bytesErr))
	return nil
}
