// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Usage:
//
//	goproxy [-listen [host]:port] [-cacheDir /tmp]
//
// goproxy serves the Go module proxy HTTP protocol at the given address (default 0.0.0.0:8081).
// It invokes the local go command to answer requests and therefore reuses
// the current GOPATH's module download cache and configuration (GOPROXY, GOSUMDB, and so on).
//
// While the proxy is running, setting GOPROXY=http://host:port will instruct the go command to use it.
// Note that the module proxy cannot share a GOPATH with its own clients or else fetches will deadlock.
// (The client will lock the entry as “being downloaded” before sending the request to the proxy,
// which will then wait for the apparently-in-progress download to finish.)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/goproxyio/goproxy/proxy"

	"golang.org/x/mod/module"
)

var downloadRoot string

const listExpire = 5 * time.Minute

var listen string
var cacheDir string
var proxyHost string
var excludeHost string

func init() {
	flag.StringVar(&excludeHost, "exclude", "", "exclude host pattern")
	flag.StringVar(&proxyHost, "proxy", "", "next hop proxy for go modules")
	flag.StringVar(&cacheDir, "cacheDir", "", "go modules cache dir")
	flag.StringVar(&listen, "listen", "0.0.0.0:8081", "service listen address")
	flag.Parse()

	if os.Getenv("GIT_TERMINAL_PROMPT") == "" {
		os.Setenv("GIT_TERMINAL_PROMPT", "0")
	}

	if os.Getenv("GIT_SSH") == "" && os.Getenv("GIT_SSH_COMMAND") == "" {
		os.Setenv("GIT_SSH_COMMAND", "ssh -o ControlMaster=no")
	}
}

func main() {
	log.SetPrefix("goproxy.io: ")
	log.SetFlags(0)
	// TODO flags
	var env struct {
		GOPATH string
	}
	if cacheDir != "" {
		downloadRoot = filepath.Join(cacheDir, "pkg/mod/cache/download")
		os.Setenv("GOPATH", cacheDir)
	}
	if err := goJSON(&env, "go", "env", "-json", "GOPATH"); err != nil {
		log.Fatal(err)
	}
	list := filepath.SplitList(env.GOPATH)
	if len(list) == 0 || list[0] == "" {
		log.Fatalf("missing $GOPATH")
	}
	downloadRoot = filepath.Join(list[0], "pkg/mod/cache/download")

	var handle http.Handler
	if proxyHost != "" {
		fmt.Fprintf(os.Stderr, "ProxyHost %s\n", proxyHost)
		if excludeHost != "" {
			fmt.Fprintf(os.Stderr, "ExcludeHost %s\n", excludeHost)
		}
		handle = &logger{proxy.NewRouter(proxy.NewServer(new(ops)), &proxy.RouterOps{
			Pattern: excludeHost,
			Proxy:   proxyHost,
		})}
	} else {
		handle = &logger{proxy.NewServer(new(ops))}
	}
	log.Fatal(http.ListenAndServe(listen, handle))
}

// goJSON runs the go command and parses its JSON output into dst.
func goJSON(dst interface{}, command ...string) error {
	cmd := exec.Command(command[0], command[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s:\n%s%s", strings.Join(command, " "), stderr.String(), stdout.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), dst); err != nil {
		return fmt.Errorf("%s: reading json: %v", strings.Join(command, " "), err)
	}
	return nil
}

// A logger is an http.Handler that logs traffic to standard error.
type logger struct {
	h http.Handler
}
type responseLogger struct {
	code int
	http.ResponseWriter
}

func (r *responseLogger) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}
func (l *logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rl := &responseLogger{code: 200, ResponseWriter: w}
	l.h.ServeHTTP(rl, r)
	fmt.Fprintf(os.Stderr, "%.3fs %d %s\n", time.Since(start).Seconds(), rl.code, r.URL)
}

// An ops is a proxy.ServerOps implementation.
type ops struct{}

func (*ops) NewContext(r *http.Request) (context.Context, error) {
	return context.Background(), nil
}
func (*ops) List(ctx context.Context, mpath string) (proxy.File, error) {
	escMod, err := module.EscapePath(mpath)
	if err != nil {
		return nil, err
	}
	file := filepath.Join(downloadRoot, escMod+"/@v/listproxy")
	if info, err := os.Stat(file); err == nil && time.Since(info.ModTime()) < listExpire {
		return os.Open(file)
	}
	var list struct {
		Path     string
		Versions []string
	}
	if err := goJSON(&list, "go", "list", "-m", "-json", "-versions", mpath+"@latest"); err != nil {
		return nil, err
	}
	if list.Path != mpath {
		return nil, fmt.Errorf("go list -m: asked for %s but got %s", mpath, list.Path)
	}
	data := []byte(strings.Join(list.Versions, "\n") + "\n")
	if len(data) == 1 {
		data = nil
	}
	os.MkdirAll(path.Dir(file), 755)
	if err := ioutil.WriteFile(file, data, 0666); err != nil {
		return nil, err
	}

	return os.Open(file)
}
func (*ops) Latest(ctx context.Context, path string) (proxy.File, error) {
	d, err := download(module.Version{Path: path, Version: "latest"})
	if err != nil {
		return nil, err
	}
	return os.Open(d.Info)
}
func (*ops) Info(ctx context.Context, m module.Version) (proxy.File, error) {
	d, err := download(m)
	if err != nil {
		return nil, err
	}
	return os.Open(d.Info)
}
func (*ops) GoMod(ctx context.Context, m module.Version) (proxy.File, error) {
	d, err := download(m)
	if err != nil {
		return nil, err
	}
	return os.Open(d.GoMod)
}
func (*ops) Zip(ctx context.Context, m module.Version) (proxy.File, error) {
	d, err := download(m)
	if err != nil {
		return nil, err
	}
	return os.Open(d.Zip)
}

type downloadInfo struct {
	Path     string
	Version  string
	Info     string
	GoMod    string
	Zip      string
	Dir      string
	Sum      string
	GoModSum string
}

func download(m module.Version) (*downloadInfo, error) {
	d := new(downloadInfo)
	return d, goJSON(d, "go", "mod", "download", "-json", m.String())
}
