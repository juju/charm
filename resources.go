// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"strings"

	"github.com/juju/schema"
)

// These are the valid resource types (except for unknown).
const (
	ResourceTypeUnknown ResourceType = ""
	ResourceTypeFile    ResourceType = "file"
)

var resourceTypes = map[ResourceType]bool{
	ResourceTypeFile: true,
}

// ResourceType enumerates the recognized resource types.
type ResourceType string

// ParseResourceType converts a string to a ResourceType. If the given
// value does not match a recognized type then ResourceTypeUnknown and
// false are returned.
func ParseResourceType(value string) (ResourceType, bool) {
	rt := ResourceType(value)
	return rt, resourceTypes[rt]
}

// String returns the printable representation of the type.
func (rt ResourceType) String() string {
	if rt == "" {
		return "<unknown>"
	}
	return string(rt)
}

// Validate ensures that the type is valid.
func (rt ResourceType) Validate() error {
	if _, ok := resourceTypes[rt]; !ok {
		return fmt.Errorf("unsupported resource type %v", rt)
	}
	return nil
}

var resourceSchema = schema.FieldMap(
	schema.Fields{
		"type":     schema.String(),
		"filename": schema.String(), // TODO(ericsnow) Change to "path"...
		"comment":  schema.String(),
	},
	schema.Defaults{
		"type":    ResourceTypeFile.String(),
		"comment": "",
	},
)

// ResourceInfo holds the information about a resource, as stored
// in a charm's metadata.
type ResourceInfo struct {
	// Name identifies the resource.
	Name string

	// Type identifies the type of resource (e.g. "file").
	Type ResourceType

	// TODO(ericsnow) Rename Path to Filename?

	// Path is the relative path of the file or directory where the
	// resource will be stored under the unit's data directory. The path
	// is resolved against a subdirectory assigned to the resource. For
	// example, given a service named "spam", a resource "eggs", and a
	// path "eggs.tgz", the fully resolved storage path for the resource
	// would be:
	//   /var/lib/juju/agent/spam-0/resources/eggs/eggs.tgz
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
		info.Type, _ = ParseResourceType(val.(string))
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

	if r.Type == ResourceTypeUnknown {
		return fmt.Errorf("resource missing type")
	}
	if err := r.Type.Validate(); err != nil {
		return fmt.Errorf("invalid resource type %v: %v", r.Type, err)
	}

	if r.Path == "" {
		// TODO(ericsnow) change "filename" to "path"
		return fmt.Errorf("resource missing filename")
	}
	if r.Type == ResourceTypeFile {
		if strings.Contains(r.Path, "/") {
			return fmt.Errorf(`filename cannot contain "/" (got %q)`, r.Path)
		}
		// TODO(ericsnow) Constrain Path to alphanumeric?
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
