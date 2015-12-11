// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource_test

import (
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable/resource"
)

var _ = gc.Suite(&ResourceSuite{})

type ResourceSuite struct{}

func (s *ResourceSuite) TestValidateFull(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Type:    resource.TypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
		Revision: 1,
	}
	err := res.Validate()

	c.Check(err, jc.ErrorIsNil)
}

func (s *ResourceSuite) TestValidateZeroValue(c *gc.C) {
	var res resource.Resource
	err := res.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
}

func (s *ResourceSuite) TestValidateBadMetadata(c *gc.C) {
	var meta resource.Meta
	c.Assert(meta.Validate(), gc.NotNil)

	res := resource.Resource{
		Meta:     meta,
		Revision: 1,
	}
	err := res.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err, gc.ErrorMatches, `.*bad metadata.*`)
}

func (s *ResourceSuite) TestValidateBadRevision(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Type:    resource.TypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
		Revision: -1,
	}
	err := res.Validate()

	c.Check(err, jc.Satisfies, errors.IsNotValid)
	c.Check(err, gc.ErrorMatches, `.*revision must be non-negative.*`)
}
