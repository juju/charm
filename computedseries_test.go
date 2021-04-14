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

func (s *computedSeriesSuite) TestCharmComputedSeriesLegacy(c *gc.C) {
	meta, err := ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
series:
  - bionic
`))
	c.Assert(err, gc.IsNil)
	dir := charmBase{
		meta:     meta,
		manifest: &Manifest{},
	}
	c.Assert(err, gc.IsNil)
	c.Assert(ComputedSeries(&dir), jc.DeepEquals, []string{"bionic"})
}

func (s *computedSeriesSuite) TestCharmComputedSeries(c *gc.C) {
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
	dir := charmBase{
		meta:     meta,
		manifest: manifest,
	}
	c.Assert(ComputedSeries(&dir), jc.DeepEquals, []string{"bionic", "focal"})
}
