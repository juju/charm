// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

// Resource is the definition for a resource that a charm uses.
type Resource struct {
	Meta

	// TODO(ericsnow) Add (e.g. "upload", "store"):
	//Origin string

	// TODO(ericsnow) Add for charm store:
	//Revision int
}

// Parse converts the provided data into a Resource.
func Parse(name string, data interface{}) Resource {
	resource := Resource{
		Meta: ParseMeta(name, data),
	}

	return resource
}

// Validate checks the payload class to ensure its data is valid.
func (res Resource) Validate() error {
	if err := res.Meta.Validate(); err != nil {
		return err
	}

	return nil
}
