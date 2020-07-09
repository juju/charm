// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"fmt"

	"github.com/juju/charm/v7"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

type OriginSuite struct{}

var _ = gc.Suite(&OriginSuite{})

func (s *OriginSuite) TestValidate(c *gc.C) {
	var originTests = []struct {
		description   string
		origin        *charm.Origin
		expectedError error
	}{
		{
			description: "local",
			origin:      &charm.Origin{Source: charm.Local},
		},
		{
			description: "charmstore",
			origin:      &charm.Origin{Source: charm.CharmStore},
		},
		{
			description: "charmhub",
			origin:      &charm.Origin{Source: charm.Charmhub},
		},
		{
			description: "unknown",
			origin:      &charm.Origin{Source: charm.Unknown},
		},
		{
			description:   "empty",
			origin:        &charm.Origin{Source: charm.Source("")},
			expectedError: fmt.Errorf(`invalid source: ""`),
		},
		{
			description:   "boom",
			origin:        &charm.Origin{Source: charm.Source("boom")},
			expectedError: fmt.Errorf(`invalid source: "boom"`),
		},
	}
	for i, test := range originTests {
		c.Logf("test %d: %s", i, test.description)
		err := test.origin.Validate()
		if err != nil {
			c.Assert(err.Error(), gc.Equals, test.expectedError.Error())
		} else {
			c.Assert(err, jc.ErrorIsNil)
		}
	}
}
