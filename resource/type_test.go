// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable/resource"
)

var _ = gc.Suite(&TypeSuite{})

type TypeSuite struct{}

func (s *TypeSuite) TestParseTypeOkay(c *gc.C) {
	rt, ok := resource.ParseType("file")

	c.Check(ok, jc.IsTrue)
	c.Check(rt, gc.Equals, resource.TypeFile)
}

func (s *TypeSuite) TestParseTypeRecognized(c *gc.C) {
	supported := []resource.Type{
		resource.TypeFile,
	}
	for _, expected := range supported {
		rt, ok := resource.ParseType(expected.String())

		c.Check(ok, jc.IsTrue)
		c.Check(rt, gc.Equals, expected)
	}
}

func (s *TypeSuite) TestParseTypeEmpty(c *gc.C) {
	rt, ok := resource.ParseType("")

	c.Check(ok, jc.IsFalse)
	c.Check(rt, gc.Equals, resource.TypeUnknown)
}

func (s *TypeSuite) TestParseTypeUnsupported(c *gc.C) {
	rt, ok := resource.ParseType("spam")

	c.Check(ok, jc.IsFalse)
	c.Check(rt, gc.Equals, resource.Type("spam"))
}

func (s *TypeSuite) TestTypeStringSupported(c *gc.C) {
	supported := map[resource.Type]string{
		resource.TypeFile: "file",
	}
	for rt, expected := range supported {
		str := rt.String()

		c.Check(str, gc.Equals, expected)
	}
}

func (s *TypeSuite) TestTypeStringUnknown(c *gc.C) {
	str := resource.TypeUnknown.String()

	c.Check(str, gc.Equals, "<unknown>")
}

func (s *TypeSuite) TestTypeStringUnsupported(c *gc.C) {
	str := resource.Type("spam").String()

	c.Check(str, gc.Equals, "spam")
}

func (s *TypeSuite) TestTypeValidateSupported(c *gc.C) {
	supported := []resource.Type{
		resource.TypeFile,
	}
	for _, rt := range supported {
		err := rt.Validate()

		c.Check(err, jc.ErrorIsNil)
	}
}

func (s *TypeSuite) TestTypeValidateUnknown(c *gc.C) {
	err := resource.TypeUnknown.Validate()

	c.Check(err, gc.ErrorMatches, `unsupported resource type .*`)
}

func (s *TypeSuite) TestTypeValidateUnsupported(c *gc.C) {
	err := resource.Type("spam").Validate()

	c.Check(err, gc.ErrorMatches, `unsupported resource type .*`)
}