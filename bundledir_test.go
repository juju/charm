// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm_test

import (
	"os"
	"path/filepath"

	"github.com/juju/testing"
	gc "launchpad.net/gocheck"

	"gopkg.in/juju/charm.v2"
	charmtesting "gopkg.in/juju/charm.v2/testing"
)

type BundleDirSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&BundleDirSuite{})

func (s *BundleDirSuite) TestReadBundleDir(c *gc.C) {
	path := charmtesting.Charms.BundleDirPath("wordpress")
	dir, err := charm.ReadBundleDir(path, func(string) error { return nil })
	c.Assert(err, gc.IsNil)
	checkWordpressBundle(c, dir, path)
}

func (s *BundleDirSuite) TestReadBundleDirWithoutREADME(c *gc.C) {
	path := charmtesting.Charms.ClonedBundleDirPath(c.MkDir(), "wordpress")
	err := os.Remove(filepath.Join(path, "README"))
	c.Assert(err, gc.IsNil)
	dir, err := charm.ReadBundleDir(path, func(string) error { return nil })
	c.Assert(err, gc.ErrorMatches, "cannot read README file: .*")
	c.Assert(dir, gc.IsNil)
}

func (s *BundleDirSuite) TestReadBundleDirWithFailedVerify(c *gc.C) {
	path := charmtesting.Charms.BundleDirPath("bad")
	dir, err := charm.ReadBundleDir(path, func(string) error { return nil })
	c.Assert(err, gc.ErrorMatches, `relation \["foo:db" "mysql:server"] refers to service "foo" not defined in this bundle`)
	c.Assert(dir, gc.IsNil)
}
