// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sumdb implements sumdb handler proxy.
package sumdb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var supportedSumDB = map[string][]string{
	"sum.golang.org":       {"https://sum.golang.org/", "https://sum.golang.google.cn/"},
	"sum.golang.google.cn": {"https://sum.golang.org/", "https://sum.golang.google.cn/"}, // db-name `sum.golang.google.cn` will be replaced in go
	"gosum.io":             {"https://gosum.io/"},
}

var (
	errSumPathInvalid = errors.New("sumdb request path invalid")
)

// Handler handles sumdb request
// goproxy.io not impl a complete sumdb, just proxy to upstream.
func Handler(w http.ResponseWriter, r *http.Request) {
	whichDB, realPath, err := parsePath(r.URL.Path)
	_, supported := supportedSumDB[whichDB]
	if err != nil || !supported {
		// if not check the target db,
		// curl https://goproxy.io/sumdb/www.google.com will succ
		w.WriteHeader(http.StatusGone)
		fmt.Fprint(w, "unsupported db")
		return
	}

	// $GOROOT/src/cmd/go/internal/modfetch/sumdb.go@initBase
	// > Before accessing any checksum database URL using a proxy, the proxy
	// > client should first fetch <proxyURL>/sumdb/<sumdb-name>/supported.
	if realPath == "supported" {
		w.WriteHeader(http.StatusOK)
		return
	}

	result := make(chan *http.Response)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second) // 2sec maybe enough
	defer cancel()

	for _, host := range supportedSumDB[whichDB] {
		go proxySumdb(ctx, host, realPath, result)
	}

	select {
	case resp := <-result:
		{
			defer resp.Body.Close()
			w.WriteHeader(resp.StatusCode)
			if _, err := io.Copy(w, resp.Body); err != nil {
				fmt.Fprint(w, err.Error())
			}
		}
	case <-ctx.Done():
		w.WriteHeader(http.StatusGone)
		fmt.Fprint(w, ctx.Err().Error())
		return
	}
}

func parsePath(rawPath string) (whichDB, path string, err error) {
	parts := strings.SplitN(rawPath, "/", 4)
	if len(parts) < 4 {
		return "", "", errSumPathInvalid
	}
	whichDB = parts[2]
	path = parts[3]
	return
}

func proxySumdb(ctx context.Context, host, path string, respChan chan<- *http.Response) {
	urlPath, err := url.Parse(host)
	if err != nil {
		return
	}
	urlPath.Path = path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlPath.String(), nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	select {
	case <-ctx.Done():
		resp.Body.Close()
	case respChan <- resp:
	}

}
