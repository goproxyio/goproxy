//go:generate bash build/generate.sh

package main

import (
	"flag"
	"github.com/goproxyio/goproxy/pkg/proxy"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var listen string
var root string

func init() {
	log.SetOutput(os.Stdout)
	flag.StringVar(&root, "root", "/go", "root cache dir to save")
	flag.StringVar(&listen, "listen", "0.0.0.0:8081", "service listen address")
	flag.Parse()
	if err := os.MkdirAll(root, os.ModePerm); err != nil {
		log.Fatalf("goproxy: make root dir failed: %s", err)
	}
}

func main() {

	// sigs := make(chan os.Signal)
	// signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("goproxy: %s inited. listen on %s\n", time.Now().Format("2006-01-02 15:04:05"), listen)

	cacheDir := filepath.Join(root, "pkg", "mod", "cache", "download")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		log.Printf("goproxy: cache dir %s is not exist. To create\n", cacheDir)
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			log.Fatalf("make cache dir failed: %s", err)
		}
	}

	http.Handle("/", proxy.NewProxy(root))
	// TODO: TLS, graceful shutdown
	err := http.ListenAndServe(listen, nil)
	if nil != err {
		panic(err)
	}
}
