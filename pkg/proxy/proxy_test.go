package proxy

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/goproxyio/goproxy/internal/modfetch"
	"github.com/goproxyio/goproxy/internal/module"
	"github.com/goproxyio/goproxy/internal/testenv"
)

var _handle http.Handler

func TestMain(m *testing.M) {
	tmpdir, err := ioutil.TempDir("", "goproxy-test-")
	if err != nil {
		log.Fatalf("init tmpdir failed: %s", err)
	}
	defer os.RemoveAll(tmpdir)
	_handle = NewProxy(tmpdir)
	m.Run()
}

var _modInfoTests = []struct {
	path    string
	query   string // query
	version string // want
	latest  bool
	time    time.Time
	gomod   string
	zip     []string
}{
	{
		path:    "gopkg.in/check.v1",
		query:   "v0.0.0-20161208181325-20d25e280405",
		version: "v0.0.0-20161208181325-20d25e280405",
		time:    time.Date(2016, 12, 8, 18, 13, 25, 0, time.UTC),
		gomod:   "module gopkg.in/check.v1\n",
		zip: []string{
			".gitignore",
			".travis.yml",
			"LICENSE",
			"README.md",
			"TODO",
			"benchmark.go",
			"benchmark_test.go",
			"bootstrap_test.go",
			"check.go",
			"check_test.go",
			"checkers.go",
			"checkers_test.go",
			"export_test.go",
			"fixture_test.go",
			"foundation_test.go",
			"helpers.go",
			"helpers_test.go",
			"printer.go",
			"printer_test.go",
			"reporter.go",
			"reporter_test.go",
			"run.go",
			"run_test.go",
		},
	},
	{
		path:    "github.com/PuerkitoBio/goquery",
		query:   "v0.0.0-20181014175806-2af3d16e2bb8",
		version: "v0.0.0-20181014175806-2af3d16e2bb8",
		time:    time.Date(2018, 10, 14, 17, 58, 6, 0, time.UTC),
		gomod:   "module github.com/PuerkitoBio/goquery\n",
		zip: []string{
			".gitattributes",
			".gitignore",
			".travis.yml",
			"LICENSE",
			"README.md",
			"array.go",
			"array_test.go",
			"bench/v0.1.0",
			"bench/v0.1.1",
			"bench/v0.1.1-v0.2.1-go1.1rc1.svg",
			"bench/v0.2.0",
			"bench/v0.2.0-v0.2.1-go1.1rc1.svg",
			"bench/v0.2.1-go1.1rc1",
			"bench/v0.3.0",
			"bench/v0.3.2-go1.2",
			"bench/v0.3.2-go1.2-take2",
			"bench/v0.3.2-go1.2rc1",
			"bench/v1.0.0-go1.7",
			"bench/v1.0.1a-go1.7",
			"bench/v1.0.1b-go1.7",
			"bench/v1.0.1c-go1.7",
			"bench_array_test.go",
			"bench_example_test.go",
			"bench_expand_test.go",
			"bench_filter_test.go",
			"bench_iteration_test.go",
			"bench_property_test.go",
			"bench_query_test.go",
			"bench_traversal_test.go",
			"doc.go",
			"doc/tips.md",
			"example_test.go",
			"expand.go",
			"expand_test.go",
			"filter.go",
			"filter_test.go",
			"iteration.go",
			"iteration_test.go",
			"manipulation.go",
			"manipulation_test.go",
			"misc/git/pre-commit",
			"property.go",
			"property_test.go",
			"query.go",
			"query_test.go",
			"testdata/gotesting.html",
			"testdata/gowiki.html",
			"testdata/metalreview.html",
			"testdata/page.html",
			"testdata/page2.html",
			"testdata/page3.html",
			"traversal.go",
			"traversal_test.go",
			"type.go",
			"type_test.go",
			"utilities.go",
			"utilities_test.go",
		},
	},
	{
		path:    "github.com/rsc/vgotest1",
		query:   "v0.0.0-20180219223237-a08abb797a67",
		version: "v0.0.0-20180219223237-a08abb797a67",
		latest:  true,
		time:    time.Date(2018, 02, 19, 22, 32, 37, 0, time.UTC),
	},
	{
		path:    "github.com/hxzhao527/legacytest",
		query:   "master",
		version: "v2.0.1+incompatible",
		time:    time.Date(2018, 07, 17, 16, 42, 53, 0, time.UTC),
		gomod:   "module github.com/hxzhao527/legacytest\n",
		zip: []string{
			"x.go",
		},
	},
	{
		path:    "github.com/micro/go-api/resolver",
		query:   "v0.5.0",
		version: "v0.5.0",
		gomod:   "module github.com/micro/go-api\n",
	},
}

var _modListTests = []struct {
	path     string
	versions []string
}{
	{
		path:     "github.com/rsc/vgotest1",
		versions: []string{"v0.0.0", "v0.0.1", "v1.0.0", "v1.0.1", "v1.0.2", "v1.0.3", "v1.1.0", "v2.0.0+incompatible"},
	},
}

func TestFetchInfo(t *testing.T) {
	testenv.MustHaveExternalNetwork(t)

	for _, mod := range _modInfoTests {
		req := buildRequest(mod.path, mod.query, ".info")

		rr, err := basicCheck(req)
		if err != nil {
			t.Error(err)
			continue
		}

		// check return data
		info := new(modfetch.RevInfo)
		if err := json.Unmarshal(rr.Body.Bytes(), info); err != nil {
			t.Errorf("package info is not recognized")
			continue
		}
		if mod.version != info.Version {
			t.Errorf("info.Version = %s, want %s", info.Version, mod.version)
		}
		if !mod.time.Equal(info.Time) {
			t.Errorf("info.Time = %v, want %v", info.Time, mod.time)
		}
	}

}
func TestFetchModFile(t *testing.T) {
	testenv.MustHaveExternalNetwork(t)

	for _, mod := range _modInfoTests {
		if len(mod.gomod) == 0 {
			continue
		}
		req := buildRequest(mod.path, mod.query, ".mod")

		rr, err := basicCheck(req)
		if err != nil {
			t.Error(err)
			continue
		}

		if data := rr.Body.String(); data != mod.gomod {
			t.Errorf("repo.GoMod(%q) = %q, want %q", mod.version, data, mod.gomod)
		}
	}
}
func TestFetchZip(t *testing.T) {
	testenv.MustHaveExternalNetwork(t)

	for _, mod := range _modInfoTests {
		if len(mod.zip) == 0 {
			continue
		}
		req := buildRequest(mod.path, mod.query, ".zip")

		rr, err := basicCheck(req)
		if err != nil {
			t.Error(err)
			continue
		}

		prefix := mod.path + "@" + mod.version + "/"
		var names []string

		data := rr.Body.Bytes() // ??
		z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			t.Errorf("open %s's zip failed: %v", mod.path, err)
			continue
		}

		for _, file := range z.File {
			if !strings.HasPrefix(file.Name, prefix) {
				t.Errorf("zip entry %v does not start with prefix %v", file.Name, prefix)
				continue
			}
			names = append(names, file.Name[len(prefix):])
		}
		if !reflect.DeepEqual(names, mod.zip) {
			t.Errorf("zip = %v\nwant %v\n", names, mod.zip)
		}
	}

}

func TestLatest(t *testing.T) {
	testenv.MustHaveExternalNetwork(t)

	for _, mod := range _modInfoTests {
		if !mod.latest {
			continue
		}
		req := buildRequest(mod.path, "latest", "")

		rr, err := basicCheck(req)
		if err != nil {
			t.Error(err)
			continue
		}

		info := new(modfetch.RevInfo)
		if err := json.Unmarshal(rr.Body.Bytes(), info); err != nil {
			t.Errorf("package info is not recognized")
			continue
		}
		if mod.version != info.Version {
			t.Errorf("info.Version = %s, want %s", info.Version, mod.version)
		}
		if !mod.time.Equal(info.Time) {
			t.Errorf("info.Time = %v, want %v", info.Time, mod.time)
		}
	}
}

func TestList(t *testing.T) {

	for _, mod := range _modListTests {
		req := buildRequest(mod.path, "", "")

		rr, err := basicCheck(req)
		if err != nil {
			t.Error(err)
			continue
		}

		modfetch.SortVersions(mod.versions)

		if data := rr.Body.String(); strings.Join(mod.versions, "\n") != data {
			t.Errorf("list not well,\n expected: %v\n, got: %v", mod.versions, strings.Split(data, "\n"))
		}
	}

}

func buildRequest(modPath, modVersion string, ext string) *http.Request {
	modPath, _ = module.EncodePath(modPath)
	modVersion, _ = module.EncodeVersion(modVersion)
	url := "/" + modPath
	switch modVersion {
	case "":
		url += "/@v/list"
	case "latest":
		url += "/@latest"
	default:
		url = url + "/@v/" + modVersion + ext
	}
	req, _ := http.NewRequest("GET", url, nil)
	return req
}

func basicCheck(req *http.Request) (*httptest.ResponseRecorder, error) {
	rr := httptest.NewRecorder()
	_handle.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		return nil, fmt.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	return rr, nil
}
