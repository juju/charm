// Copyright 2019 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"sort"
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

func (*bundleDataOverlaySuite) TestExtractBaseAndOverlayPartsWithNoOverlayFields(c *gc.C) {
	data := `
bundle: kubernetes
applications:
  mysql:
    charm: cs:mysql
    scale: 1
  wordpress:
    charm: cs:wordpress
    scale: 2
relations:
- - wordpress:db
  - mysql:mysql
`

	expBase := `
bundle: kubernetes
applications:
  mysql:
    charm: cs:mysql
    series: kubernetes
    num_units: 1
  wordpress:
    charm: cs:wordpress
    series: kubernetes
    num_units: 2
relations:
- - wordpress:db
  - mysql:mysql
`

	expOverlay := `
{}
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

func (*bundleDataOverlaySuite) TestVerifyNoOverlayFieldsPresent(c *gc.C) {
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

	bd, err := charm.ReadBundleData(strings.NewReader(data))
	c.Assert(err, gc.IsNil)

	static, overlay, err := charm.ExtractBaseAndOverlayParts(bd)
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(charm.VerifyNoOverlayFieldsPresent(static), gc.Equals, nil)

	expErrors := []string{
		"applications.apache2.offers can only appear in an overlay section",
		"applications.apache2.offers.my-offer.endpoints can only appear in an overlay section",
		"applications.apache2.offers.my-offer.acl can only appear in an overlay section",
		"applications.apache2.offers.my-other-offer.endpoints can only appear in an overlay section",
	}
	err = charm.VerifyNoOverlayFieldsPresent(overlay)
	c.Assert(err, gc.FitsTypeOf, (*charm.VerificationError)(nil))
	errors := err.(*charm.VerificationError).Errors
	errStrings := make([]string, len(errors))
	for i, err := range errors {
		errStrings[i] = err.Error()
	}
	sort.Strings(errStrings)
	sort.Strings(expErrors)
	c.Assert(errStrings, jc.DeepEquals, expErrors)
}
