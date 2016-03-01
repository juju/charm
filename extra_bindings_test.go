// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"fmt"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable"
)

var _ = gc.Suite(&extraBindingsSuite{})

type extraBindingsSuite struct{}

func (s *extraBindingsSuite) TestSchemaOkay(c *gc.C) {
	raw := map[interface{}]interface{}{
		"foo": nil,
		"bar": nil,
	}
	v, err := charm.ExtraBindingsSchema.Coerce(raw, nil)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(v, jc.DeepEquals, map[interface{}]interface{}{
		"foo": nil,
		"bar": nil,
	})
}

func (s *extraBindingsSuite) TestSchemaValuesMustBeEmpty(c *gc.C) {
	badValues := []interface{}{
		42, true, 3.14, "bad", []string{"a"}, map[string]string{"x": "y"},
	}
	for _, testValue := range badValues {
		raw := map[interface{}]interface{}{
			"some-endpoint": testValue,
		}
		v, err := charm.ExtraBindingsSchema.Coerce(raw, nil)
		expectedError := fmt.Sprintf("some-endpoint: expected empty value, got %T(%#v)", testValue, testValue)
		c.Check(err, gc.NotNil)
		c.Check(err.Error(), gc.Equals, expectedError)
		c.Check(v, gc.IsNil)
	}
}

func (s *extraBindingsSuite) TestValidateWithEmptyNonNilMap(c *gc.C) {
	err := charm.ValidateMetaExtraBindings(map[string]charm.ExtraBinding{})
	c.Assert(err, gc.ErrorMatches, "extra bindings cannot be empty when specified")
}

func (s *extraBindingsSuite) TestValidateWithEmptyName(c *gc.C) {
	invalidBindings := map[string]charm.ExtraBinding{
		"": charm.ExtraBinding{Name: ""},
	}
	err := charm.ValidateMetaExtraBindings(invalidBindings)
	c.Assert(err, gc.ErrorMatches, "missing extra binding name")
}

func (s *extraBindingsSuite) TestValidateWithMismatchedName(c *gc.C) {
	invalidBindings := map[string]charm.ExtraBinding{
		"bar": charm.ExtraBinding{Name: "foo"},
	}
	err := charm.ValidateMetaExtraBindings(invalidBindings)
	c.Assert(err, gc.ErrorMatches, `mismatched extra binding name: got "foo", expected "bar"`)
}
