// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Package proxy implements the HTTP protocols for serving a Go module proxy.
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/mod/module"
)

// A ServerOps provides the external operations
// (accessing module information and so on) needed by the Server.
type ServerOps interface {
	// NewContext returns the context to use for the request r.
	NewContext(r *http.Request) (context.Context, error)
	// List, Latest, Info, GoMod, and Zip all return a File to be sent to a client.
	// The File will be closed after its contents are sent.
	// In the case of an error, if the error satisfies errors.Is(err, os.ErrNotFound),
	// the server responds with an HTTP 404 error;
	// otherwise it responds with an HTTP 500 error.
	// List returns a list of tagged versions of the module identified by path.
	// The versions should all be canonical semantic versions
	// and formatted in a text listing, one per line.
	// Pseudo-versions derived from untagged commits should be omitted.
	// The go command exposes this list in 'go list -m -versions' output
	// and also uses it to resolve wildcards like 'go get m@v1.2'.
	List(ctx context.Context, path string) (File, error)
	// Latest returns an info file for the latest known version of the module identified by path.
	// The go command uses this for 'go get m' or 'go get m@latest'
	// but only after finding no suitable version among the ones returned by List.
	// Typically, Latest should return a pseudo-version for the latest known commit.
	Latest(ctx context.Context, path string) (File, error)
	// Info opens and returns the module version's info file.
	// The requested version can be a canonical semantic version
	// but can also be an arbitrary version reference, like "master".
	//
	// The metadata in the returned file should be a JSON object corresponding
	// to the Go type
	//
	//	type Info struct {
	//		Version string
	//		Time time.Time
	//	}
	//
	// where the version is the resolved canonical semantic version
	// and the time is the commit or publication time of that version
	// (for use with go list -m).
	// The NewInfo function can be used to construct an info File.
	//
	// Proxies should obtain the module version information by
	// executing 'go mod download -json' and caching the file
	// listed in the Info field.
	Info(ctx context.Context, m module.Version) (File, error)
	// GoMod opens and returns the module's go.mod file.
	// The requested version is a canonical semantic version.
	//
	// Proxies should obtain the module version information by
	// executing 'go mod download -json' and caching the file
	// listed in the GoMod field.
	GoMod(ctx context.Context, m module.Version) (File, error)
	// Zip opens and returns the module's zip file.
	// The requested version is a canonical semantic version.
	//
	// Proxies should obtain the module version information by
	// executing 'go mod download -json' and caching the file
	// listed in the Zip field.
	Zip(ctx context.Context, m module.Version) (File, error)
}

// A File is a file to be served, typically an *os.File or the result of calling MemFile or NewInfo.
// The modification time is the only necessary field in the Stat result.
type File interface {
	io.Reader
	io.Seeker
	io.Closer
	Stat() (os.FileInfo, error)
}

// A Server is the proxy HTTP server,
// which implements http.Handler and should be invoked
// to serve the paths listed in ServerPaths.
//
// The server assumes that the requests are made to the root of the URL space,
// so it should typically be registered using:
//
//	srv := proxy.NewServer(ops)
//	http.Handle("/", srv)
//
// To register a server at a subdirectory of the URL space, wrap the server in http.StripPrefix:
//
//	srv := proxy.NewServer(ops)
//	http.Handle("/proxy/", http.StripPrefix("/proxy", srv))
//
// All recognized requests to the server contain the substring "/@v/" in the URL.
// The server will respond with an http.StatusBadRequest (400) error to unrecognized requests.
type Server struct {
	ops ServerOps
}

// NewServer returns a new Server using the given operations.
func NewServer(ops ServerOps) *Server {
	return &Server{ops: ops}
}

// ServeHTTP is the server's implementation of http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.ops.NewContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	i := strings.Index(r.URL.Path, "/@")
	if i < 0 {
		http.Error(w, "no such path", http.StatusNotFound)
		return
	}
	modPath, err := module.UnescapePath(strings.TrimPrefix(r.URL.Path[:i], "/"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	what := r.URL.Path[i+len("/@"):]
	const (
		contentTypeJSON   = "application/json"
		contentTypeText   = "text/plain; charset=UTF-8"
		contentTypeBinary = "application/octet-stream"
	)
	var ctype string
	var f File
	var openErr error
	switch what {
	case "latest":
		ctype = contentTypeJSON
		f, openErr = s.ops.Latest(ctx, modPath)
	case "v/list":
		ctype = contentTypeText
		f, openErr = s.ops.List(ctx, modPath)
	default:
		what = strings.TrimPrefix(what, "v/")
		ext := path.Ext(what)
		vers, err := module.UnescapeVersion(strings.TrimSuffix(what, ext))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		m := module.Version{Path: modPath, Version: vers}
		if vers == "latest" {
			// The go command handles "go get m@latest" by fetching /m/@v/latest, not latest.info.
			// We should never see requests for "latest.info" and so on, so avoid confusion
			// by disallowing it early.
			http.Error(w, "version latest is disallowed", http.StatusNotFound)
			return
		}
		// All requests require canonical versions except for info,
		// which accepts any revision identifier known to the underlying storage.
		if ext != ".info" && vers != module.CanonicalVersion(vers) {
			http.Error(w, "version "+vers+" is not in canonical form", http.StatusNotFound)
			return
		}
		switch ext {
		case ".info":
			ctype = "application/json"
			f, openErr = s.ops.Info(ctx, m)
		case ".mod":
			ctype = "text/plain; charset=UTF-8"
			f, openErr = s.ops.GoMod(ctx, m)
		case ".zip":
			ctype = "application/octet-stream"
			f, openErr = s.ops.Zip(ctx, m)
		default:
			http.Error(w, "request not recognized", http.StatusNotFound)
			return
		}
	}
	if openErr != nil {
		code := http.StatusNotFound
		http.Error(w, openErr.Error(), code)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "unexpected directory", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", ctype)
	http.ServeContent(w, r, what, info.ModTime(), f)
}

// MemFile returns an File containing the given in-memory content and modification time.
func MemFile(data []byte, t time.Time) File {
	return &memFile{bytes.NewReader(data), memStat{t, int64(len(data))}}
}

type memFile struct {
	*bytes.Reader
	stat memStat
}

func (f *memFile) Close() error                             { return nil }
func (f *memFile) Stat() (os.FileInfo, error)               { return &f.stat, nil }
func (f *memFile) Readdir(count int) ([]os.FileInfo, error) { return nil, os.ErrInvalid }

type memStat struct {
	t    time.Time
	size int64
}

func (s *memStat) Name() string       { return "memfile" }
func (s *memStat) Size() int64        { return s.size }
func (s *memStat) Mode() os.FileMode  { return 0444 }
func (s *memStat) ModTime() time.Time { return s.t }
func (s *memStat) IsDir() bool        { return false }
func (s *memStat) Sys() interface{}   { return nil }

// NewInfo returns a formatted info file for the given version, time pair.
// The version should be a canonical semantic version.
func NewInfo(version string, t time.Time) File {
	var info = struct {
		Version string
		Time    time.Time
	}{version, t}
	js, err := json.Marshal(info)
	if err != nil {
		// json.Marshal only fails for bad types; there are no bad types in info.
		panic("unexpected json.Marshal failure")
	}
	return MemFile(js, t)
}
