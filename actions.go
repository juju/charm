// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/juju/gojsonschema"
	goyaml "gopkg.in/yaml.v1"
)

var prohibitedSchemaKeys = map[string]bool{"$ref": true, "$schema": true}

var actionNameRule = regexp.MustCompile("^[a-z](?:[a-z-]*[a-z])?$")

// Actions defines the available actions for the charm.  Additional params
// may be added as metadata at a future time (e.g. version.)
type Actions struct {
	ActionSpecs map[string]ActionSpec `yaml:"actions,omitempty" bson:",omitempty"`
}

// ActionSpec is a definition of the parameters and traits of an Action.
// The Params map is expected to conform to JSON-Schema Draft 4 as defined at
// http://json-schema.org/draft-04/schema# (see http://json-schema.org/latest/json-schema-core.html)
type ActionSpec struct {
	Description string
	Params      map[string]interface{}
}

func NewActions() *Actions {
	return &Actions{}
}

// ValidateParams tells us whether an unmarshaled JSON object conforms to the
// Params for the specific ActionSpec.
// Usage: ok, err := ch.Actions()["snapshot"].Validate(jsonParams)
func (spec *ActionSpec) ValidateParams(params interface{}) (bool, error) {

	specSchemaDoc, err := gojsonschema.NewJsonSchemaDocument(spec.Params)
	if err != nil {
		return false, err
	}

	results := specSchemaDoc.Validate(params)
	if results.Valid() {
		return true, nil
	}

	var errorStrings []string
	for _, validationError := range results.Errors() {
		errorStrings = append(errorStrings, validationError.String())
	}
	return false, fmt.Errorf("JSON validation failed: %s", strings.Join(errorStrings, "; "))
}

// ReadActions builds an Actions spec from a charm's actions.yaml.
func ReadActionsYaml(r io.Reader) (*Actions, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var unmarshaledActions Actions
	if err := goyaml.Unmarshal(data, &unmarshaledActions); err != nil {
		return nil, err
	}

	for name, actionSpec := range unmarshaledActions.ActionSpecs {
		if valid := actionNameRule.MatchString(name); !valid {
			return nil, fmt.Errorf("bad action name %s", name)
		}

		// Clean any map[interface{}]interface{}s out so they don't
		// cause problems with BSON serialization later.
		cleansedParams, err := cleanse(actionSpec.Params)
		if err != nil {
			return nil, err
		}

		// JSON-Schema must be a map
		cleansedParamsMap, ok := cleansedParams.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("the params failed to parse as a map")
		}

		// Now substitute the cleansed map into the original.
		var tempSpec = unmarshaledActions.ActionSpecs[name]
		tempSpec.Params = cleansedParamsMap
		unmarshaledActions.ActionSpecs[name] = tempSpec

		// Make sure the new Params doc conforms to JSON-Schema
		// Draft 4 (http://json-schema.org/latest/json-schema-core.html)
		_, err = gojsonschema.NewJsonSchemaDocument(unmarshaledActions.ActionSpecs[name].Params)
		if err != nil {
			return nil, fmt.Errorf("invalid params schema for action schema %s: %v", name, err)
		}

	}
	return &unmarshaledActions, nil
}

// cleanse rejects schemas containing references or maps keyed with non-
// strings, and coerces acceptable maps to contain only maps with string keys.
func cleanse(input interface{}) (interface{}, error) {
	switch typedInput := input.(type) {

	// In this case, recurse in.
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for key, value := range typedInput {

			if prohibitedSchemaKeys[key] {
				return nil, fmt.Errorf("schema key %q not compatible with this version of juju", key)
			}

			newValue, err := cleanse(value)
			if err != nil {
				return nil, err
			}
			newMap[key] = newValue
		}
		return newMap, nil

	// Coerce keys to strings and error out if there's a problem; then recurse.
	case map[interface{}]interface{}:
		newMap := make(map[string]interface{})
		for key, value := range typedInput {
			typedKey, ok := key.(string)
			if !ok {
				return nil, errors.New("map keyed with non-string value")
			}
			newMap[typedKey] = value
		}
		return cleanse(newMap)

	// Recurse
	case []interface{}:
		newSlice := make([]interface{}, 0)
		for _, sliceValue := range typedInput {
			newSliceValue, err := cleanse(sliceValue)
			if err != nil {
				return nil, errors.New("map keyed with non-string value")
			}
			newSlice = append(newSlice, newSliceValue)
		}
		return newSlice, nil

	// Other kinds of values are OK.
	default:
		return input, nil
	}
}
