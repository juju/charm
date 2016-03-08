// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

// Resource is the definition for a resource that a charm uses.
type Resource struct {
	Info

	// TODO(ericsnow) Add (e.g. "upload", "store"):
	//Origin string

	// TODO(ericsnow) Add for charm store:
	//Revision int
}

// Parse converts the provided data into a Resource.
func Parse(name string, data interface{}) Resource {
	resource := Resource{
		Info: parseInfo(name, data),
	}

	return resource
}

// Validate checks the payload class to ensure its data is valid.
func (r Resource) Validate() error {
	if err := r.Info.Validate(); err != nil {
		return err
	}

	return nil
}
