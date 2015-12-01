// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

import (
	"fmt"
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

// ParseType converts a string to a Type. If the given
// value does not match a recognized type then TypeUnknown and
// false are returned.
func ParseType(value string) (Type, bool) {
	rt := Type(value)
	return rt, types[rt]
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
		return fmt.Errorf("unsupported resource type %v", rt)
	}
	return nil
}
