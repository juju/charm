// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable/resource"
)

var _ = gc.Suite(&ResourceSuite{})

type ResourceSuite struct{}

func (s *ResourceSuite) TestParseOkay(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	res := resource.Parse(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Type:    resource.TypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	})
}

func (s *ResourceSuite) TestParseMissingName(c *gc.C) {
	name := ""
	data := map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	res := resource.Parse(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		Meta: resource.Meta{
			Name:    "",
			Type:    resource.TypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	})
}

func (s *ResourceSuite) TestParseMissingType(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	res := resource.Parse(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Type:    resource.TypeUnknown,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	})
}

func (s *ResourceSuite) TestParseMissingPath(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"type":    "file",
		"comment": "One line that is useful when operators need to push it.",
	}
	res := resource.Parse(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Type:    resource.TypeFile,
			Path:    "",
			Comment: "One line that is useful when operators need to push it.",
		},
	})
}

func (s *ResourceSuite) TestParseMissingComment(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
	}
	res := resource.Parse(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Type:    resource.TypeFile,
			Path:    "filename.tgz",
			Comment: "",
		},
	})
}

func (s *ResourceSuite) TestParseEmpty(c *gc.C) {
	name := "my-resource"
	data := make(map[string]interface{})
	res := resource.Parse(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		Meta: resource.Meta{
			Name: "my-resource",
		},
	})
}

func (s *ResourceSuite) TestParseNil(c *gc.C) {
	name := "my-resource"
	var data map[string]interface{}
	res := resource.Parse(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		Meta: resource.Meta{
			Name: "my-resource",
		},
	})
}

func (s *ResourceSuite) TestValidateFull(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Type:    resource.TypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, jc.ErrorIsNil)
}

func (s *ResourceSuite) TestValidateZeroValue(c *gc.C) {
	var res resource.Resource
	err := res.Validate()

	c.Check(err, gc.NotNil)
}

func (s *ResourceSuite) TestValidateMissingName(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Type:    resource.TypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing name`)
}

func (s *ResourceSuite) TestValidateMissingType(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing type`)
}

func (s *ResourceSuite) TestValidateUnknownType(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Type:    "repo",
			Path:    "repo-root",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `.*unsupported resource type .*`)
}

func (s *ResourceSuite) TestValidateMissingPath(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name:    "my-resource",
			Type:    resource.TypeFile,
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing filename`)
}

func (s *ResourceSuite) TestValidateNestedPath(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name: "my-resource",
			Type: resource.TypeFile,
			Path: "spam/eggs",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `.*filename cannot contain "/" .*`)
}

func (s *ResourceSuite) TestValidateAbsolutePath(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name: "my-resource",
			Type: resource.TypeFile,
			Path: "/spam/eggs",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `.*filename cannot contain "/" .*`)
}

func (s *ResourceSuite) TestValidateSuspectPath(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name: "my-resource",
			Type: resource.TypeFile,
			Path: "git@github.com:juju/juju.git",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `.*filename cannot contain "/" .*`)
}

func (s *ResourceSuite) TestValidateMissingComment(c *gc.C) {
	res := resource.Resource{
		Meta: resource.Meta{
			Name: "my-resource",
			Type: resource.TypeFile,
			Path: "filename.tgz",
		},
	}
	err := res.Validate()

	c.Check(err, jc.ErrorIsNil)
}
