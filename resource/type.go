// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

import (
	"github.com/juju/errors"
)

// These are the valid resource types (except for unknown).
var (
	TypeFile = Type{"file"}
)

var types = map[Type]bool{
	TypeFile: true,
}

// Type enumerates the recognized resource types.
type Type struct {
	str string
}

// ParseType converts a string to a Type. If the given value does not
// match a recognized type then an error is returned.
func ParseType(value string) (Type, error) {
	for rt := range types {
		if value == rt.str {
			return rt, nil
		}
	}
	return Type{}, errors.Errorf("unsupported resource type %q", value)
}

// String returns the printable representation of the type.
func (rt Type) String() string {
	return rt.str
}

// Validate ensures that the type is valid.
func (rt Type) Validate() error {
	// Only the zero value is invalid.
	var zero Type
	if rt == zero {
		return errors.NotValidf("zero value")
	}
	return nil
}
