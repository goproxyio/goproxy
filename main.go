//go:generate ./build/generate.sh

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/goproxyio/goproxy/pkg/proxy"
)

const (
	VERSION = "1.0.0"
)

var (
	listen   string
	cacheDir string
	version  bool
)

func init() {
	log.SetOutput(os.Stdout)
	flag.BoolVar(&version, "v", false, "display the version")
	flag.StringVar(&cacheDir, "cache_dir", "", "go modules cache dir")
	flag.StringVar(&listen, "listen", "0.0.0.0:8081", "service listen address")
	flag.Parse()
}

// TODO
// 1.Standardize log's formatting
// 2.Add http timeout setting
// 3.Fix the locking problem : cache folder was still locked after the process stopped
func main() {
	errCh := make(chan error)

	if version {
		fmt.Printf("goproxy version is %s \n", VERSION)
		os.Exit(0)
		return
	}

	log.Printf("goproxy: %s inited. listen on %s\n", time.Now().Format("2006-01-02 15:04:05"), listen)

	if cacheDir == "" {
		cacheDir = "/go"
		gpEnv := os.Getenv("GOPATH")
		if gpEnv != "" {
			gp := filepath.SplitList(gpEnv)
			if gp[0] != "" {
				cacheDir = gp[0]
			}
		}
	}
	fullCacheDir := filepath.Join(cacheDir, "pkg", "mod", "cache", "download")
	if _, err := os.Stat(fullCacheDir); os.IsNotExist(err) {
		log.Printf("goproxy: cache dir %s is not exist. To create it.\n", fullCacheDir)
		if err := os.MkdirAll(fullCacheDir, 0755); err != nil {
			log.Fatalf("goproxy: make cache dir failed: %s", err)
		}
	}
	server := http.Server{
		Addr:    listen,
		Handler: proxy.NewProxy(cacheDir),
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			errCh <- err
		}
	}()

	signCh := make(chan os.Signal)
	signal.Notify(signCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Fatal(err)
	case sign := <-signCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		log.Printf("goproxy: Server gracefully %s", sign)
	}
}
