// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"

	"github.com/juju/schema"
)

// These are the valid resource types.
const (
	ResourceTypeFile = "file"
)

var resourceTypes = map[string]bool{
	ResourceTypeFile: true,
}

func isValidResourceType(value string) bool {
	return resourceTypes[value]
}

var resourceSchema = schema.FieldMap(
	schema.Fields{
		"type":     schema.String(),
		"filename": schema.String(), // TODO(ericsnow) Change to "path"...
		"comment":  schema.String(),
	},
	schema.Defaults{
		"type":    ResourceTypeFile,
		"comment": "",
	},
)

// ResourceInfo holds the information about a resource, as stored
// in a charm's metadata.
type ResourceInfo struct {
	// Name identifies the resource.
	Name string

	// Type identifies the type of resource (e.g. "file").
	Type string

	// Path is where the resource will be stored.
	Path string

	// Comment holds optional user-facing info for the resource.
	Comment string
}

func parseResourceInfo(name string, data interface{}) ResourceInfo {
	var info ResourceInfo
	info.Name = name

	if data == nil {
		return info
	}
	rMap := data.(map[string]interface{})

	if val := rMap["type"]; val != nil {
		info.Type = val.(string)
	}

	if val := rMap["filename"]; val != nil {
		info.Path = val.(string)
	}

	if val := rMap["comment"]; val != nil {
		info.Comment = val.(string)
	}

	return info
}

// Validate checks the payload class to ensure its data is valid.
func (r ResourceInfo) Validate() error {
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

// Resource is the definition for a resource that a charm uses.
type Resource struct {
	ResourceInfo

	// TODO(ericsnow) Add (e.g. "upload", "store"):
	//Origin string

	// TODO(ericsnow) Add for charm store:
	//Revision int
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
		ResourceInfo: parseResourceInfo(name, data),
	}

	return resource
}

// Validate checks the payload class to ensure its data is valid.
func (r Resource) Validate() error {
	if err := r.ResourceInfo.Validate(); err != nil {
		return err
	}

	return nil
}
