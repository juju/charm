// Copyright 2019 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"strings"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	yaml "gopkg.in/yaml.v2"

	"gopkg.in/juju/charm.v6"
)

type bundleDataOverlaySuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&bundleDataOverlaySuite{})

func (*bundleDataOverlaySuite) TestExtractBaseAndOverlayParts(c *gc.C) {
	data := `
applications:
  apache2:
    charm: cs:apache2-26
    offers:
      my-offer:
        endpoints:
        - apache-website
        - website-cache
        acl:
          admin: admin
          foo: consume
      my-other-offer:
        endpoints:
        - apache-website
saas:
    apache2:
        url: production:admin/info.apache
series: bionic
`

	expBase := `
applications:
  apache2:
    charm: cs:apache2-26
saas:
  apache2:
    url: production:admin/info.apache
series: bionic
`

	expOverlay := `
applications:
  apache2:
    offers:
      my-offer:
        endpoints:
        - apache-website
        - website-cache
        acl:
          admin: admin
          foo: consume
      my-other-offer:
        endpoints:
        - apache-website
`

	bd, err := charm.ReadBundleData(strings.NewReader(data))
	c.Assert(err, gc.IsNil)

	base, overlay, err := charm.ExtractBaseAndOverlayParts(bd)
	c.Assert(err, jc.ErrorIsNil)

	baseYaml, err := yaml.Marshal(base)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert("\n"+string(baseYaml), gc.Equals, expBase)

	overlayYaml, err := yaml.Marshal(overlay)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert("\n"+string(overlayYaml), gc.Equals, expOverlay)
}
