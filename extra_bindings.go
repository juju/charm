// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"reflect"
	"strings"

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
var extraBindingsSchema = schema.Map(nonEmptyStringC{"binding name"}, noValueC{})

// noValueC is a schema.Checker that only succeeds if the input an empty
// value (nil), but unlike the equivalent schema.Const(nil), noValueC provides a
// slightly better error message.
type noValueC struct{}

func (c noValueC) Coerce(v interface{}, path []string) (interface{}, error) {
	if reflect.DeepEqual(v, nil) {
		return v, nil
	}
	pathPrefix := schemaPathAsPrefix(path)
	return nil, fmt.Errorf("%sexpected no value, got %T(%#v)", pathPrefix, v, v)
}

// TODO(dimitern): Move noValueC and nonEmptyStringC into the schema package and
// remove this helper, copied from schema.pathAsPrefix.
func schemaPathAsPrefix(path []string) string {
	if len(path) == 0 {
		return ""
	}
	var s string
	if path[0] == "." {
		s = strings.Join(path[1:], "")
	} else {
		s = strings.Join(path, "")
	}
	if s == "" {
		return ""
	}
	return s + ": "
}

// nonEmptyStringC is a schema.Checker that only succeeds if the input is a
// non-empty string. To tweak the error message, valueLable can contain a
// singular label of the value being checked, or "string" will be used when
// valueLabel is "".
type nonEmptyStringC struct {
	valueLabel string
}

func (c nonEmptyStringC) Coerce(v interface{}, path []string) (interface{}, error) {
	stringValue, err := schema.String().Coerce(v, path)
	if err != nil {
		return nil, err
	}
	if stringValue.(string) == "" {
		pathPrefix := schemaPathAsPrefix(path)
		label := c.valueLabel
		if label == "" {
			label = "string"
		}
		return nil, fmt.Errorf("%sexpected non-empty %s, got \"\"", pathPrefix, label)

	}
	return stringValue, nil
}

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
