package proxy

import (
	"github.com/goproxyio/goproxy/pkg/modfetch"
	"github.com/goproxyio/goproxy/pkg/modfetch/codehost"
	"github.com/goproxyio/goproxy/pkg/module"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	tmpdir, err := ioutil.TempDir("", "goproxy-test-")
	if err != nil {
		log.Fatalf("init tmpdir failed: %s", err)
	}
	defer os.RemoveAll(tmpdir)
	modfetch.PkgMod = filepath.Join(tmpdir, "pkg/mod")
	codehost.WorkRoot = filepath.Join(modfetch.PkgMod, "cache/vcs")
	os.Exit(m.Run())
}

func TestFetchInfo(t *testing.T) {
	packagePath := "gopkg.in/check.v1"
	version := "v0.0.0-20161208181325-20d25e280405"
	info, err := modfetch.InfoFile(packagePath, version)
	if err != nil {
		t.Errorf("fetch %s@%s info get error: %s", packagePath, version, err)
	}
	t.Logf("%s@%s info on %s", packagePath, version, info)
}
func TestFetchModFile(t *testing.T) {
	packagePath := "gopkg.in/check.v1"
	version := "v0.0.0-20161208181325-20d25e280405"
	info, err := modfetch.GoModFile(packagePath, version)
	if err != nil {
		t.Errorf("fetch %s@%s modfile get error: %s", packagePath, version, err)
	}
	t.Logf("%s@%s modfile on %s", packagePath, version, info)
}
func TestFetchModSum(t *testing.T) {
	packagePath := "gopkg.in/check.v1"
	version := "v0.0.0-20161208181325-20d25e280405"
	info, err := modfetch.GoModSum(packagePath, version)
	if err != nil {
		t.Errorf("fetch %s@%s modsum get error: %s", packagePath, version, err)
	}
	t.Logf("%s@%s modsum is %s", packagePath, version, info)
}
func TestFetchZip(t *testing.T) {
	packagePath := "gopkg.in/check.v1"
	version := "v0.0.0-20161208181325-20d25e280405"
	mod := module.Version{Path: packagePath, Version: version}
	info, err := modfetch.DownloadZip(mod)
	if err != nil {
		t.Errorf("fetch %s@%s modsum get error: %s", packagePath, version, err)
	}
	t.Logf("%s@%s modsum on %s", packagePath, version, info)
}
func TestDownload(t *testing.T) {
	packagePath := "gopkg.in/check.v1"
	version := "v0.0.0-20161208181325-20d25e280405"
	mod := module.Version{Path: packagePath, Version: version}
	info, err := modfetch.Download(mod)
	if err != nil {
		t.Errorf("fetch %s@%s modsum get error: %s", packagePath, version, err)
	}
	t.Logf("%s@%s modsum on %s", packagePath, version, info)
}

func TestLatest(t *testing.T) {
	packagePath := "golang.org/x/net"
	version := "latest"
	repo, err := modfetch.Lookup(packagePath)
	if err != nil {
		t.Errorf("lookup %s get error %s", packagePath, err)
	}
	info, err := repo.Latest()
	if err != nil {
		t.Errorf("fetch %s@%s info get error %s", packagePath, version, err)
	}
	t.Logf("%s@%s info is %s", packagePath, version, info)
}

func TestList(t *testing.T) {
	packagePath := "golang.org/x/net"
	version := "latest"
	repo, err := modfetch.Lookup(packagePath)
	if err != nil {
		t.Errorf("lookup %s get error %s", packagePath, err)
	}
	info, err := repo.Versions("")
	if err != nil {
		t.Errorf("fetch %s@%s versions get error %s", packagePath, version, err)
	}
	t.Logf("%s@%s versions are %s", packagePath, version, info)
}
