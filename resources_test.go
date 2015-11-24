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

func (s *resourceSuite) TestParseResourceOkay(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"type":     "file",
		"filename": "filename.tgz",
		"comment":  "One line that is useful when operators need to push it.",
	}
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		Name:     "my-resource",
		Type:     "file",
		Filename: "filename.tgz",
		Comment:  "One line that is useful when operators need to push it.",
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
		Name:     "",
		Type:     "file",
		Filename: "filename.tgz",
		Comment:  "One line that is useful when operators need to push it.",
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
		Name:     "my-resource",
		Type:     "",
		Filename: "filename.tgz",
		Comment:  "One line that is useful when operators need to push it.",
	})
}

func (s *resourceSuite) TestParseResourceMissingFilename(c *gc.C) {
	name := "my-resource"
	data := map[string]interface{}{
		"type":    "file",
		"comment": "One line that is useful when operators need to push it.",
	}
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		Name:     "my-resource",
		Type:     "file",
		Filename: "",
		Comment:  "One line that is useful when operators need to push it.",
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
		Name:     "my-resource",
		Type:     "file",
		Filename: "filename.tgz",
		Comment:  "",
	})
}

func (s *resourceSuite) TestParseResourceEmpty(c *gc.C) {
	name := "my-resource"
	data := make(map[string]interface{})
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		Name: "my-resource",
	})
}

func (s *resourceSuite) TestParseResourceNil(c *gc.C) {
	name := "my-resource"
	var data map[string]interface{}
	resource := charm.ParseResource(name, data)

	c.Check(resource, jc.DeepEquals, charm.Resource{
		Name: "my-resource",
	})
}

func (s *resourceSuite) TestValidateFull(c *gc.C) {
	resource := charm.Resource{
		Name:     "my-resource",
		Type:     "file",
		Filename: "filename.tgz",
		Comment:  "One line that is useful when operators need to push it.",
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
		Type:     "file",
		Filename: "filename.tgz",
		Comment:  "One line that is useful when operators need to push it.",
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing name`)
}

func (s *resourceSuite) TestValidateMissingType(c *gc.C) {
	resource := charm.Resource{
		Name:     "my-resource",
		Filename: "filename.tgz",
		Comment:  "One line that is useful when operators need to push it.",
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing type`)
}

func (s *resourceSuite) TestValidateUnknownType(c *gc.C) {
	resource := charm.Resource{
		Name:     "my-resource",
		Type:     "repo",
		Filename: "git@github.com:juju/juju.git",
		Comment:  "One line that is useful when operators need to push it.",
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `unrecognized resource type .*`)
}

func (s *resourceSuite) TestValidateMissingFilename(c *gc.C) {
	resource := charm.Resource{
		Name:    "my-resource",
		Type:    "file",
		Comment: "One line that is useful when operators need to push it.",
	}
	err := resource.Validate()

	c.Check(err, gc.ErrorMatches, `resource missing filename`)
}

func (s *resourceSuite) TestValidateMissingComment(c *gc.C) {
	resource := charm.Resource{
		Name:     "my-resource",
		Type:     "file",
		Filename: "filename.tgz",
	}
	err := resource.Validate()

	c.Check(err, jc.ErrorIsNil)
}
