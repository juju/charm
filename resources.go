// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"

	"github.com/juju/schema"
	"github.com/juju/utils/set"
)

const (
	resourceTypeFile = "file"
)

var resourceTypes = set.NewStrings(
	resourceTypeFile,
)

func isValidResourceType(value string) bool {
	return resourceTypes.Contains(value)
}

var resourceSchema = schema.FieldMap(
	schema.Fields{
		"type":     schema.String(),
		"filename": schema.String(), // TODO(ericsnow) Change to "path"...
		"comment":  schema.String(),
	},
	schema.Defaults{
		"comment": "",
	},
)

// Resource holds the information about a resource, as stored
// in a charm's metadata.
type Resource struct {
	// Name identifies the resource.
	Name string

	// Type identifies the type of resource (e.g. "file").
	Type string

	// Path is where the resource will be stored.
	Path string

	// Comment holds optional user-facing info for the resource.
	Comment string
}

func parseResources(data interface{}) map[string]Resource {
	if data == nil {
		return nil
	}

	result := make(map[string]Resource)
	for name, val := range data.(map[string]interface{}) {
		result[name] = parseResource(name, val)
	}

	return result
}

func parseResource(name string, data interface{}) Resource {
	resource := Resource{
		Name: name,
	}
	if data == nil {
		return resource
	}
	rMap := data.(map[string]interface{})

	if val := rMap["type"]; val != nil {
		resource.Type = val.(string)
	}

	if val := rMap["filename"]; val != nil {
		resource.Path = val.(string)
	}

	if val := rMap["comment"]; val != nil {
		resource.Comment = val.(string)
	}

	return resource
}

// Validate checks the payload class to ensure its data is valid.
func (r Resource) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("resource missing name")
	}

	if r.Type == "" {
		return fmt.Errorf("resource missing type")
	}
	if !isValidResourceType(r.Type) {
		return fmt.Errorf("unrecognized resource type %q", r.Type)
	}

	if r.Path == "" {
		// TODO(ericsnow) change "filename" to "path"
		return fmt.Errorf("resource missing filename")
	}

	return nil
}
