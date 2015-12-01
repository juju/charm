// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable"
)

var _ = gc.Suite(&resourceSuite{})

type resourceSuite struct{}

func (s *resourceSuite) TestSchemaOkay(c *gc.C) {
	raw := map[interface{}]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	v, err := charm.ResourceSchema.Coerce(raw, nil)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(v, jc.DeepEquals, map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	})
}

func (s *resourceSuite) TestSchemaMissingType(c *gc.C) {
	raw := map[interface{}]interface{}{
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	v, err := charm.ResourceSchema.Coerce(raw, nil)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(v, jc.DeepEquals, map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	})
}

func (s *resourceSuite) TestSchemaUnknownType(c *gc.C) {
	raw := map[interface{}]interface{}{
		"type":     "repo",
		"filename": "juju",
		"comment":  "One line that is useful when operators need to push it.",
	}
	v, err := charm.ResourceSchema.Coerce(raw, nil)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(v, jc.DeepEquals, map[string]interface{}{
		"type":     "repo",
		"filename": "juju",
		"comment":  "One line that is useful when operators need to push it.",
	})
}

func (s *resourceSuite) TestSchemaMissingPath(c *gc.C) {
	raw := map[interface{}]interface{}{
		"type":    "file",
		"comment": "One line that is useful when operators need to push it.",
	}
	_, err := charm.ResourceSchema.Coerce(raw, nil)

	c.Check(err, gc.NotNil)
}

func (s *resourceSuite) TestSchemaMissingComment(c *gc.C) {
	raw := map[interface{}]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
	}
	v, err := charm.ResourceSchema.Coerce(raw, nil)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(v, jc.DeepEquals, map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "",
	})
}

func (s *resourceSuite) TestParseResourceOkay(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name:    "my-resource",
			Type:    charm.ResourceTypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	})
}

func (s *resourceSuite) TestParseResourceMissingName(c *gc.C) {
	name := ""
	data := map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name:    "",
			Type:    charm.ResourceTypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	})
}

func (s *resourceSuite) TestParseResourceMissingType(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name:    "my-resource",
			Type:    charm.ResourceTypeUnknown,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	})
}

func (s *resourceSuite) TestParseResourceMissingPath(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"type":    "file",
		"comment": "One line that is useful when operators need to push it.",
	}
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name:    "my-resource",
			Type:    charm.ResourceTypeFile,
			Path:    "",
			Comment: "One line that is useful when operators need to push it.",
		},
	})
}

func (s *resourceSuite) TestParseResourceMissingComment(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
	}
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name:    "my-resource",
			Type:    charm.ResourceTypeFile,
			Path:    "filename.tgz",
			Comment: "",
		},
	})
}

func (s *resourceSuite) TestParseResourceEmpty(c *gc.C) {
	name := "my-resource"
	data := make(map[string]interface{})
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name: "my-resource",
		},
	})
}

func (s *resourceSuite) TestParseResourceNil(c *gc.C) {
	name := "my-resource"
	var data map[string]interface{}
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name: "my-resource",
		},
	})
}

func (s *resourceSuite) TestValidateFull(c *gc.C) {
	resource := charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name:    "my-resource",
			Type:    charm.ResourceTypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := resource.Validate()

	c.Check(err, jc.ErrorIsNil)
}

func (s *resourceSuite) TestValidateZeroValue(c *gc.C) {
	var resource charm.Resource
	err := resource.Validate()

	c.Check(err, gc.NotNil)
}

func (s *resourceSuite) TestValidateMissingName(c *gc.C) {
	resource := charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Type:    charm.ResourceTypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing name`)
}

func (s *resourceSuite) TestValidateMissingType(c *gc.C) {
	resource := charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name:    "my-resource",
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing type`)
}

func (s *resourceSuite) TestValidateUnknownType(c *gc.C) {
	resource := charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name:    "my-resource",
			Type:    "repo",
			Path:    "repo-root",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `.*unsupported resource type .*`)
}

func (s *resourceSuite) TestValidateMissingPath(c *gc.C) {
	resource := charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name:    "my-resource",
			Type:    charm.ResourceTypeFile,
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing filename`)
}

func (s *resourceSuite) TestValidateNestedPath(c *gc.C) {
	resource := charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name: "my-resource",
			Type: charm.ResourceTypeFile,
			Path: "spam/eggs",
		},
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `.*filename cannot contain "/" .*`)
}

func (s *resourceSuite) TestValidateAbsolutePath(c *gc.C) {
	resource := charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name: "my-resource",
			Type: charm.ResourceTypeFile,
			Path: "/spam/eggs",
		},
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `.*filename cannot contain "/" .*`)
}

func (s *resourceSuite) TestValidateSuspectPath(c *gc.C) {
	resource := charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name: "my-resource",
			Type: charm.ResourceTypeFile,
			Path: "git@github.com:juju/juju.git",
		},
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `.*filename cannot contain "/" .*`)
}

func (s *resourceSuite) TestValidateMissingComment(c *gc.C) {
	resource := charm.Resource{
		ResourceInfo: charm.ResourceInfo{
			Name: "my-resource",
			Type: charm.ResourceTypeFile,
			Path: "filename.tgz",
		},
	}
	err := resource.Validate()

	c.Check(err, jc.ErrorIsNil)
}
