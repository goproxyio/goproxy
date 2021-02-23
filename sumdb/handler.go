// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sumdb implements sumdb handler proxy.
package sumdb

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var enableGoogleSumDB bool
var supportedSumDB = []string{
	"sum.golang.org",
	"gosum.io",
}

func init() {
	// 启动 sumdb，在真实请求时会使用 goproxy.cn 代理该请求
	enableGoogleSumDB = true
	// go func() {
	// 	p := "https://sum.golang.org"
	// 	_, err := http.Get(p)
	// 	if err == nil {
	// 		enableGoogleSumDB = true
	// 	}
	// }()
}

//Handler handles sumdb request
func Handler(w http.ResponseWriter, r *http.Request) {
	if !enableGoogleSumDB {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/supported") {
		for _, supported := range supportedSumDB {
			uri := fmt.Sprintf("/sumdb/%s/supported", supported)
			if r.URL.Path == uri {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		w.WriteHeader(http.StatusGone)
		return
	}
	var p string
	// 使用 proxy.cn 代理 supportedSumDB
	if isSupportedSumDB(r.URL.Path) {
		p = "https://goproxy.cn" + r.URL.Path
		log.Printf("Proxy sum db with goproxy.cn, url: %s\n", p)
	} else {
		p = "https://" + strings.TrimPrefix(r.URL.Path, "/sumdb/")
	}
	_, err := url.Parse(p)
	if err != nil {
		w.WriteHeader(http.StatusGone)
		fmt.Fprintf(w, err.Error())
		return
	}

	resp, err := http.Get(p)
	if err != nil {
		w.WriteHeader(http.StatusGone)
		fmt.Fprintf(w, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	return
}

func isSupportedSumDB(hostPath string) bool {
	for _, sumdbHost := range supportedSumDB {
		if strings.Contains(hostPath, sumdbHost) {
			return true
		}
	}
	return false
}
