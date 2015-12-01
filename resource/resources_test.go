// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable/resource"
)

var _ = gc.Suite(&resourceSuite{})

type resourceSuite struct{}

func (s *resourceSuite) TestParseResourceTypeOkay(c *gc.C) {
	rt, ok := resource.ParseResourceType("file")

	c.Check(ok, jc.IsTrue)
	c.Check(rt, gc.Equals, resource.ResourceTypeFile)
}

func (s *resourceSuite) TestParseResourceTypeRecognized(c *gc.C) {
	supported := []resource.ResourceType{
		resource.ResourceTypeFile,
	}
	for _, expected := range supported {
		rt, ok := resource.ParseResourceType(expected.String())

		c.Check(ok, jc.IsTrue)
		c.Check(rt, gc.Equals, expected)
	}
}

func (s *resourceSuite) TestParseResourceTypeEmpty(c *gc.C) {
	rt, ok := resource.ParseResourceType("")

	c.Check(ok, jc.IsFalse)
	c.Check(rt, gc.Equals, resource.ResourceTypeUnknown)
}

func (s *resourceSuite) TestParseResourceTypeUnsupported(c *gc.C) {
	rt, ok := resource.ParseResourceType("spam")

	c.Check(ok, jc.IsFalse)
	c.Check(rt, gc.Equals, resource.ResourceType("spam"))
}

func (s *resourceSuite) TestResourceTypeStringSupported(c *gc.C) {
	supported := map[resource.ResourceType]string{
		resource.ResourceTypeFile: "file",
	}
	for rt, expected := range supported {
		str := rt.String()

		c.Check(str, gc.Equals, expected)
	}
}

func (s *resourceSuite) TestResourceTypeStringUnknown(c *gc.C) {
	str := resource.ResourceTypeUnknown.String()

	c.Check(str, gc.Equals, "<unknown>")
}

func (s *resourceSuite) TestResourceTypeStringUnsupported(c *gc.C) {
	str := resource.ResourceType("spam").String()

	c.Check(str, gc.Equals, "spam")
}

func (s *resourceSuite) TestResourceTypeValidateSupported(c *gc.C) {
	supported := []resource.ResourceType{
		resource.ResourceTypeFile,
	}
	for _, rt := range supported {
		err := rt.Validate()

		c.Check(err, jc.ErrorIsNil)
	}
}

func (s *resourceSuite) TestResourceTypeValidateUnknown(c *gc.C) {
	err := resource.ResourceTypeUnknown.Validate()

	c.Check(err, gc.ErrorMatches, `unsupported resource type .*`)
}

func (s *resourceSuite) TestResourceTypeValidateUnsupported(c *gc.C) {
	err := resource.ResourceType("spam").Validate()

	c.Check(err, gc.ErrorMatches, `unsupported resource type .*`)
}

func (s *resourceSuite) TestParseResourceOkay(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	res := resource.ParseResource(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name:    "my-resource",
			Type:    resource.ResourceTypeFile,
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
	res := resource.ParseResource(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name:    "",
			Type:    resource.ResourceTypeFile,
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
	res := resource.ParseResource(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name:    "my-resource",
			Type:    resource.ResourceTypeUnknown,
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
	res := resource.ParseResource(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name:    "my-resource",
			Type:    resource.ResourceTypeFile,
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
	res := resource.ParseResource(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name:    "my-resource",
			Type:    resource.ResourceTypeFile,
			Path:    "filename.tgz",
			Comment: "",
		},
	})
}

func (s *resourceSuite) TestParseResourceEmpty(c *gc.C) {
	name := "my-resource"
	data := make(map[string]interface{})
	res := resource.ParseResource(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name: "my-resource",
		},
	})
}

func (s *resourceSuite) TestParseResourceNil(c *gc.C) {
	name := "my-resource"
	var data map[string]interface{}
	res := resource.ParseResource(name, data)

	c.Check(res, jc.DeepEquals, resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name: "my-resource",
		},
	})
}

func (s *resourceSuite) TestValidateFull(c *gc.C) {
	res := resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name:    "my-resource",
			Type:    resource.ResourceTypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, jc.ErrorIsNil)
}

func (s *resourceSuite) TestValidateZeroValue(c *gc.C) {
	var res resource.Resource
	err := res.Validate()

	c.Check(err, gc.NotNil)
}

func (s *resourceSuite) TestValidateMissingName(c *gc.C) {
	res := resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Type:    resource.ResourceTypeFile,
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing name`)
}

func (s *resourceSuite) TestValidateMissingType(c *gc.C) {
	res := resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name:    "my-resource",
			Path:    "filename.tgz",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing type`)
}

func (s *resourceSuite) TestValidateUnknownType(c *gc.C) {
	res := resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name:    "my-resource",
			Type:    "repo",
			Path:    "repo-root",
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `.*unsupported resource type .*`)
}

func (s *resourceSuite) TestValidateMissingPath(c *gc.C) {
	res := resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name:    "my-resource",
			Type:    resource.ResourceTypeFile,
			Comment: "One line that is useful when operators need to push it.",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing filename`)
}

func (s *resourceSuite) TestValidateNestedPath(c *gc.C) {
	res := resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name: "my-resource",
			Type: resource.ResourceTypeFile,
			Path: "spam/eggs",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `.*filename cannot contain "/" .*`)
}

func (s *resourceSuite) TestValidateAbsolutePath(c *gc.C) {
	res := resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name: "my-resource",
			Type: resource.ResourceTypeFile,
			Path: "/spam/eggs",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `.*filename cannot contain "/" .*`)
}

func (s *resourceSuite) TestValidateSuspectPath(c *gc.C) {
	res := resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name: "my-resource",
			Type: resource.ResourceTypeFile,
			Path: "git@github.com:juju/juju.git",
		},
	}
	err := res.Validate()

	c.Check(err, gc.ErrorMatches, `.*filename cannot contain "/" .*`)
}

func (s *resourceSuite) TestValidateMissingComment(c *gc.C) {
	res := resource.Resource{
		ResourceInfo: resource.ResourceInfo{
			Name: "my-resource",
			Type: resource.ResourceTypeFile,
			Path: "filename.tgz",
		},
	}
	err := res.Validate()

	c.Check(err, jc.ErrorIsNil)
}
