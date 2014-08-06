// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm_test

import (
	"os"
	"path/filepath"

	"github.com/juju/testing"
	"gopkg.in/juju/charm.v3"
	charmtesting "gopkg.in/juju/charm.v3/testing"
	gc "launchpad.net/gocheck"
)

type BundleDirSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&BundleDirSuite{})

func (s *BundleDirSuite) TestReadBundleDir(c *gc.C) {
	path := charmtesting.Charms.BundleDirPath("wordpress")
	dir, err := charm.ReadBundleDir(path)
	c.Assert(err, gc.IsNil)
	checkWordpressBundle(c, dir, path)
}

func (s *BundleDirSuite) TestReadBundleDirWithoutREADME(c *gc.C) {
	path := charmtesting.Charms.ClonedBundleDirPath(c.MkDir(), "wordpress")
	err := os.Remove(filepath.Join(path, "README.md"))
	c.Assert(err, gc.IsNil)
	dir, err := charm.ReadBundleDir(path)
	c.Assert(err, gc.ErrorMatches, "cannot read README file: .*")
	c.Assert(dir, gc.IsNil)
}

func (s *BundleDirSuite) TestArchiveTo(c *gc.C) {
	baseDir := c.MkDir()
	charmDir := charmtesting.Charms.ClonedBundleDirPath(baseDir, "wordpress")
	s.assertArchiveTo(c, baseDir, charmDir)
}

func (s *BundleDirSuite) assertArchiveTo(c *gc.C, baseDir, bundleDir string) {
	dir, err := charm.ReadBundleDir(bundleDir)
	c.Assert(err, gc.IsNil)
	path := filepath.Join(baseDir, "archive.bundle")
	file, err := os.Create(path)
	c.Assert(err, gc.IsNil)
	err = dir.ArchiveTo(file)
	file.Close()
	c.Assert(err, gc.IsNil)

	archive, err := charm.ReadBundleArchive(path)
	c.Assert(err, gc.IsNil)
	c.Assert(archive.ReadMe(), gc.Equals, dir.ReadMe())
	c.Assert(archive.Data(), gc.DeepEquals, dir.Data())
}
