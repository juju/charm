// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm_test

import (
	"fmt"
	gc "launchpad.net/gocheck"
	"os"
	"path/filepath"

	"gopkg.in/juju/charm.v3"
	charmtesting "gopkg.in/juju/charm.v3/testing"
)

var _ = gc.Suite(&BundleArchiveSuite{})

type BundleArchiveSuite struct {
	archivePath string
}

func (s *BundleArchiveSuite) SetUpSuite(c *gc.C) {
	s.archivePath = charmtesting.Charms.BundleArchivePath(c.MkDir(), "wordpress")
}

func (s *BundleArchiveSuite) TestReadBundleArchive(c *gc.C) {
	archive, err := charm.ReadBundleArchive(s.archivePath, verifyOk)
	c.Assert(err, gc.IsNil)
	checkWordpressBundle(c, archive, s.archivePath)
}

func (s *BundleArchiveSuite) TestReadBundleArchiveWithoutBundleYAML(c *gc.C) {
	testReadBundleArchiveWithoutFile(c, "bundle.yaml")
}

func (s *BundleArchiveSuite) TestReadBundleArchiveWithoutREADME(c *gc.C) {
	testReadBundleArchiveWithoutFile(c, "README.md")
}

func testReadBundleArchiveWithoutFile(c *gc.C, fileToRemove string) {
	path := charmtesting.Charms.ClonedBundleDirPath(c.MkDir(), "wordpress")
	dir, err := charm.ReadBundleDir(path, verifyOk)
	c.Assert(err, gc.IsNil)

	// Remove the file from the bundle directory.
	// ArchiveTo just zips the contents of the directory as-is,
	// so the resulting bundle archive not contain the
	// file.
	err = os.Remove(filepath.Join(dir.Path, fileToRemove))
	c.Assert(err, gc.IsNil)

	archivePath := filepath.Join(c.MkDir(), "out.bundle")
	dstf, err := os.Create(archivePath)
	c.Assert(err, gc.IsNil)

	err = dir.ArchiveTo(dstf)
	dstf.Close()

	archive, err := charm.ReadBundleArchive(archivePath, verifyOk)
	// Slightly dubious assumption: the quoted file name has no
	// regexp metacharacters worth worrying about.
	c.Assert(err, gc.ErrorMatches, fmt.Sprintf("archive file %q not found", fileToRemove))
	c.Assert(archive, gc.IsNil)
}

func (s *BundleArchiveSuite) TestExpandTo(c *gc.C) {
	dir := c.MkDir()
	archive, err := charm.ReadBundleArchive(s.archivePath, verifyOk)
	c.Assert(err, gc.IsNil)
	err = archive.ExpandTo(dir)
	c.Assert(err, gc.IsNil)
	bdir, err := charm.ReadBundleDir(dir, verifyOk)
	c.Assert(err, gc.IsNil)
	c.Assert(bdir.ReadMe(), gc.Equals, archive.ReadMe())
	c.Assert(bdir.Data(), gc.DeepEquals, archive.Data())
}
