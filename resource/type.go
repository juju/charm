// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

import (
	"fmt"

	"github.com/juju/errors"
)

// These are the valid resource types (except for unknown).
const (
	TypeUnknown Type = ""
	TypeFile    Type = "file"
)

var types = map[Type]bool{
	TypeFile: true,
}

// Type enumerates the recognized resource types.
type Type string

// ParseType converts a string to a Type. If the given value does not
// match a recognized type then an error is returned.
func ParseType(value string) (Type, error) {
	rt := Type(value)
	return rt, rt.Validate()
}

// String returns the printable representation of the type.
func (rt Type) String() string {
	if rt == "" {
		return "<unknown>"
	}
	return string(rt)
}

// Validate ensures that the type is valid.
func (rt Type) Validate() error {
	if _, ok := types[rt]; !ok {
		msg := fmt.Sprintf("unsupported resource type %v", rt)
		return errors.NewNotValid(nil, msg)
	}
	return nil
}
