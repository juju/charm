// Copyright 2019 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/juju/testing"
	gc "gopkg.in/check.v1"
)

type BundleDataSourceSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&BundleDataSourceSuite{})

func (s *BundleDataSourceSuite) TestReadBundleFromLocalFile(c *gc.C) {
	path := bundleDirPath(c, "wordpress-multidoc")
	src, err := LocalBundleDataSource(filepath.Join(path, "bundle.yaml"))
	c.Assert(err, gc.IsNil)
	assertBundleSourceProcessed(c, src)
}

func (s *BundleDataSourceSuite) TestReadBundleFromExplodedArchiveFolder(c *gc.C) {
	path := bundleDirPath(c, "wordpress-multidoc")
	src, err := LocalBundleDataSource(path)
	c.Assert(err, gc.IsNil)
	assertBundleSourceProcessed(c, src)
}

func (s *BundleDataSourceSuite) TestReadBundleFromArchive(c *gc.C) {
	path := archiveBundleDirPath(c, "wordpress-multidoc")
	src, err := LocalBundleDataSource(path)
	c.Assert(err, gc.IsNil)
	assertBundleSourceProcessed(c, src)
}

func (s *BundleDataSourceSuite) TestReadBundleFromStream(c *gc.C) {
	r := strings.NewReader(`
applications:
  wordpress:
    charm: wordpress
  mysql:
    charm: mysql
    num_units: 1
relations:
  - ["wordpress:db", "mysql:server"]
--- # overlay.yaml
applications:
  wordpress:
    offers:
      offer1:
        endpoints:
          - "some-endpoint"
--- # overlay2.yaml
applications:
  wordpress:
    offers:
      offer1:
        acl:
          admin: "admin"
          foo: "consume"
`)

	src, err := StreamBundleDataSource(r, "https://example.com")
	c.Assert(err, gc.IsNil)
	assertBundleSourceProcessed(c, src)
}

func assertBundleSourceProcessed(c *gc.C, src BundleDataSource) {
	parts := src.Parts()
	c.Assert(parts, gc.HasLen, 3)
	assertFieldPresent(c, parts[1], "applications.wordpress.offers.offer1.endpoints")
	assertFieldPresent(c, parts[2], "applications.wordpress.offers.offer1.acl.admin")
}

func assertFieldPresent(c *gc.C, part *BundleDataPart, path string) {
	var (
		segments             = strings.Split(path, ".")
		next     interface{} = part.PresenseMap
	)

	for segIndex, segment := range segments {
		c.Assert(next, gc.NotNil, gc.Commentf("incomplete path: %s", strings.Join(segments[:segIndex], ".")))
		switch typ := next.(type) {
		case FieldPresenseMap:
			next = typ[segment]
			c.Assert(next, gc.NotNil, gc.Commentf("incomplete path: %s", strings.Join(segments[:segIndex+1], ".")))
		default:
			c.Fatalf("unexpected type %T at path: %s", typ, strings.Join(segments[:segIndex], "."))
		}
	}
}

func bundleDirPath(c *gc.C, name string) string {
	path := filepath.Join("internal/test-charm-repo/bundle", name)
	assertIsDir(c, path)
	return path
}

func assertIsDir(c *gc.C, path string) {
	info, err := os.Stat(path)
	c.Assert(err, gc.IsNil)
	c.Assert(info.IsDir(), gc.Equals, true)
}

func archiveBundleDirPath(c *gc.C, name string) string {
	src := filepath.Join("internal/test-charm-repo/bundle", name, "bundle.yaml")
	srcYaml, err := ioutil.ReadFile(src)
	c.Assert(err, gc.IsNil)

	dstPath := filepath.Join(c.MkDir(), "bundle.zip")
	f, err := os.Create(dstPath)
	c.Assert(err, gc.IsNil)
	defer func() { c.Assert(f.Close(), gc.IsNil) }()

	zw := zip.NewWriter(f)
	defer func() { c.Assert(zw.Close(), gc.IsNil) }()
	w, err := zw.Create("bundle.yaml")
	c.Assert(err, gc.IsNil)
	_, err = w.Write(srcYaml)
	c.Assert(err, gc.IsNil)

	return dstPath
}
