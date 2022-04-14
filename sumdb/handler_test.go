// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sumdb implements sumdb handler proxy.
package sumdb

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	if ret := t.Run("supported", testSupported); !ret {
		t.Error("supported test failed, stop test")
		t.FailNow()
	}
	t.Run("proxy", testProxy)
}

func testSupported(t *testing.T) {
	type TestCase struct {
		name          string
		db            string
		wantSupported bool
	}

	tests := []TestCase{
		{
			name:          "sum.golang.org",
			db:            "sum.golang.org",
			wantSupported: true,
		},
		{
			name:          "gosum.io",
			db:            "gosum.io",
			wantSupported: true,
		},
		{
			name:          "sum.golang.google.cn",
			db:            "sum.golang.google.cn",
			wantSupported: true,
		},
		{
			name:          "other",
			db:            "other",
			wantSupported: false,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			recoder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("https://goproxy.io/sumdb/%s/supported", testcase.db), nil)
			Handler(recoder, req)

			resp := recoder.Result()
			if support := (resp.StatusCode == http.StatusOK); support != testcase.wantSupported {
				t.Errorf("db %s: want %v got %v", testcase.db, testcase.wantSupported, support)
			}
			resp.Body.Close()
		})
	}
}

func testProxy(t *testing.T) {
	type TestCase struct {
		name       string
		db         string
		path       string
		expectSucc bool
	}
	tests := []TestCase{
		{
			name:       "lookup",
			db:         "sum.golang.google.cn",
			path:       "lookup/github.com/goproxyio/goproxy@v1.0.0", // this is a fake testcase
			expectSucc: true,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {

			recoder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("https://goproxy.io/sumdb/%s/%s", testcase.db, testcase.path), nil)
			Handler(recoder, req)

			resp := recoder.Result()
			if succ := (resp.StatusCode == http.StatusOK); succ != testcase.expectSucc {
				t.Errorf("FETCH from db %s/%s got unexpect http status %d", testcase.db, testcase.path, resp.StatusCode)
				return
			}
		})
	}
}
