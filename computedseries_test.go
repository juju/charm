// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"strings"
)

type computedSeriesSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&computedSeriesSuite{})

func (s *computedSeriesSuite) TestDirComputedSeriesLegacy(c *gc.C) {
	meta, err := ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
series:
  - bionic
`))
	c.Assert(err, gc.IsNil)
	dir := CharmDir{
		meta:     meta,
		manifest: &Manifest{},
	}
	c.Assert(err, gc.IsNil)
	c.Assert(dir.ComputedSeries(), jc.DeepEquals, []string{"bionic"})
}

func (s *computedSeriesSuite) TestDirComputedSeries(c *gc.C) {
	meta, err := ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
`))
	c.Assert(err, gc.IsNil)
	manifest, err := ReadManifest(strings.NewReader(`
bases:
  - name: ubuntu
    channel: "18.04"
  - name: ubuntu
    channel: "20.04"
`))
	c.Assert(err, gc.IsNil)
	dir := CharmDir{
		meta:     meta,
		manifest: manifest,
	}
	c.Assert(dir.ComputedSeries(), jc.DeepEquals, []string{"bionic", "focal"})
}

func (s *computedSeriesSuite) TestArchiveComputedSeriesLegacy(c *gc.C) {
	meta, err := ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
series:
  - bionic
`))
	c.Assert(err, gc.IsNil)
	arc := CharmArchive{
		meta:     meta,
		manifest: &Manifest{},
	}
	c.Assert(arc.ComputedSeries(), jc.DeepEquals, []string{"bionic"})
}

func (s *computedSeriesSuite) TestArchiveComputedSeries(c *gc.C) {
	meta, err := ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
`))
	c.Assert(err, gc.IsNil)
	manifest, err := ReadManifest(strings.NewReader(`
bases:
  - name: ubuntu
    channel: 14.04/stable
`))
	c.Assert(err, gc.IsNil)
	arc := CharmArchive{
		meta:     meta,
		manifest: manifest,
	}
	c.Assert(arc.ComputedSeries(), jc.DeepEquals, []string{"trusty"})
}
