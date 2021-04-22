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
    architectures: ["amd64","aarch64","s390x"]
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
		Architectures: []string{"amd64", "arm64", "s390x"},
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

func (s *manifestSuite) TestReadValidateManifest(c *gc.C) {
	_, err := ReadManifest(strings.NewReader(`
bases:
  - name: ""
    channel: "18.04"
`))
	c.Assert(err, gc.ErrorMatches, "manifest: name must be specified not valid")
}

func (s *manifestSuite) TestValidateManifest(c *gc.C) {
	manifest := &Manifest{
		Bases: []Base{{
			Name: "",
		}},
	}
	c.Assert(manifest.Validate(), gc.ErrorMatches, "invalid base: name must be specified not valid")
}
