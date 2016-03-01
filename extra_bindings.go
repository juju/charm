// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"

	"github.com/juju/schema"
)

// ExtraBinding represents a bindable endpoint which is not also a relation.
type ExtraBinding struct {
	Name string `bson:"name" json:"Name"`
}

// When specified, the "extra-bindings" section in the metadata.yaml
// should have the following format:
//
// extra-bindings:
//     "<endpoint-name>":
//     ...
// Endpoint names are strings and may match an existing relation name from
// the Provides, Requires, or Peers metadata sections. The values beside
// each endpoint name must be left out (i.e. "foo": <anything> is invalid).
var extraBindingsSchema = schema.Map(schema.NonEmptyString("binding name"), schema.Nil(""))

func parseMetaExtraBindings(data interface{}) (map[string]ExtraBinding, error) {
	if data == nil {
		return nil, nil
	}

	bindingsMap := data.(map[interface{}]interface{})
	result := make(map[string]ExtraBinding)
	for name, _ := range bindingsMap {
		stringName := name.(string)
		result[stringName] = ExtraBinding{Name: stringName}
	}

	return result, nil
}

func validateMetaExtraBindings(extraBindings map[string]ExtraBinding) error {
	if extraBindings == nil {
		return nil
	} else if len(extraBindings) == 0 {
		return fmt.Errorf("extra bindings cannot be empty when specified")
	}

	for name, binding := range extraBindings {
		if binding.Name == "" || name == "" {
			return fmt.Errorf("missing extra binding name")
		}
		if binding.Name != name {
			return fmt.Errorf("mismatched extra binding name: got %q, expected %q", binding.Name, name)
		}
	}
	return nil
}
