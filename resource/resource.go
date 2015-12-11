// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

import (
	"github.com/juju/errors"
)

// Resource describes a charm's resource in the charm store.
type Resource struct {
	Meta

	// Revision is the charm store revision of the resource.
	Revision int
}

// Validate checks the payload class to ensure its data is valid.
func (r Resource) Validate() error {
	if err := r.Meta.Validate(); err != nil {
		return errors.Annotate(err, "bad metadata")
	}

	if r.Revision < 0 {
		return errors.NewNotValid(nil, "invalid resource (revision must be non-negative)")
	}

	return nil
}
