package main

import (
	"bytes"
	"flag"
	"fmt"
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
			if strings.HasSuffix(r.URL.Path, ".info") {
				mod := strings.Split(r.URL.Path, "/@v/")
				if len(mod) != 2 {
					ReturnServerError(w, fmt.Errorf("bad module path:%s", r.URL.Path))
					return
				}
				version := strings.TrimSuffix(mod[1], ".info")
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
				stdout, stderr, err := goGet(path + "@" + version)
				if err != nil {
					ReturnServerError(w, fmt.Errorf("stdout: %s stderr: %s", stdout, stderr))
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

func goGet(path string) (string, string, error) {
	fmt.Fprintf(os.Stdout, "goproxy: download %s\n", path)
	cmd := exec.Command("go", "get", "-d", path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return string(stdout.Bytes()), string(stderr.Bytes()), err
}
