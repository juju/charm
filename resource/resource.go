// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

import (
	"github.com/juju/errors"
)

// Resource describes a charm's resource in the charm store.
type Resource struct {
	Meta

	// Origin identifies where the resource will come from.
	Origin Origin

	// Revision is the charm store revision of the resource.
	Revision int

	// Fingerprint is the SHA-384 checksum for the resource blob.
	Fingerprint Fingerprint

	// Size is the size of the resource, in bytes.
	Size int64
}

// Validate checks the payload class to ensure its data is valid.
func (res Resource) Validate() error {
	if err := res.Meta.Validate(); err != nil {
		return errors.Annotate(err, "invalid resource (bad metadata)")
	}

	if err := res.Origin.Validate(); err != nil {
		return errors.Annotate(err, "invalid resource (bad origin)")
	}

	if res.Revision < 0 {
		return errors.NewNotValid(nil, "invalid resource (revision must be non-negative)")
	}
	// TODO(ericsnow) Ensure Revision is 0 for OriginUpload?

	if res.Fingerprint.IsZero() {
		if res.Size > 0 {
			return errors.NewNotValid(nil, "missing fingerprint")
		}
	} else {
		if err := res.Fingerprint.Validate(); err != nil {
			return errors.Annotate(err, "bad fingerprint")
		}
	}

	if res.Size < 0 {
		return errors.NotValidf("negative size")
	}

	return nil
}
