// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

import (
	"fmt"
	"strings"
)

// Info holds the information about a resource, as stored
// in a charm's metadata.
type Info struct {
	// Name identifies the resource.
	Name string

	// Type identifies the type of resource (e.g. "file").
	Type Type

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

func parseInfo(name string, data interface{}) Info {
	var info Info
	info.Name = name

	if data == nil {
		return info
	}
	rMap := data.(map[string]interface{})

	if val := rMap["type"]; val != nil {
		info.Type, _ = ParseType(val.(string))
	}

	if val := rMap["filename"]; val != nil {
		info.Path = val.(string)
	}

	if val := rMap["comment"]; val != nil {
		info.Comment = val.(string)
	}

	return info
}

// Validate checks the resource info to ensure the data is valid.
func (r Info) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("resource missing name")
	}

	if r.Type == TypeUnknown {
		return fmt.Errorf("resource missing type")
	}
	if err := r.Type.Validate(); err != nil {
		return fmt.Errorf("invalid resource type %v: %v", r.Type, err)
	}

	if r.Path == "" {
		// TODO(ericsnow) change "filename" to "path"
		return fmt.Errorf("resource missing filename")
	}
	if r.Type == TypeFile {
		if strings.Contains(r.Path, "/") {
			return fmt.Errorf(`filename cannot contain "/" (got %q)`, r.Path)
		}
		// TODO(ericsnow) Constrain Path to alphanumeric?
	}

	return nil
}
