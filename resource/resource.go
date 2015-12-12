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

	// Fingerprint is the SHA-384 checksum for the resource blob.
	Fingerprint Fingerprint
}

// Validate checks the payload class to ensure its data is valid.
func (res Resource) Validate() error {
	if err := res.Meta.Validate(); err != nil {
		return errors.Annotate(err, "invalid resource (bad metadata)")
	}

	if res.Revision < 0 {
		return errors.NewNotValid(nil, "invalid resource (revision must be non-negative)")
	}

	if err := res.Fingerprint.Validate(); err != nil {
		return errors.Annotate(err, "bad fingerprint")
	}

	return nil
}
