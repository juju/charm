// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"strings"

	"github.com/juju/testing"
	gc "gopkg.in/check.v1"
)

type manifestSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&manifestSuite{})

func (s *manifestSuite) TestReadManifest(c *gc.C) {
	manifest, err := ReadManifest(strings.NewReader(`
bases:
  - name: ubuntu
    channel: "18.04"
  - name: ubuntu
    channel: "20.04/stable"
`))
	c.Assert(err, gc.IsNil)
	c.Assert(manifest, gc.DeepEquals, &Manifest{[]Base{{
		Name: "ubuntu",
		Channel: Channel{
			Track:  "18.04",
			Risk:   "stable",
			Branch: "",
		},
	}, {
		Name: "ubuntu",
		Channel: Channel{
			Track:  "20.04",
			Risk:   "stable",
			Branch: "",
		},
	},
	}})
}
