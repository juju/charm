// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

import (
	"crypto/sha512"
	"fmt"

	"github.com/juju/errors"
)

const fingerprintSize = 48 // 384 / 8

// Fingerprint represents the unique fingerprint value of a resource's data.
type Fingerprint struct {
	raw []byte
}

// NewFingerprint returns a resource fingerprint using the provded hash.
// The data is not checked for correctness.
func NewFingerprint(raw []byte) Fingerprint {
	fp := Fingerprint{
		raw: raw,
	}

	return fp
}

// BuildFingerprint returns the resource fingerprint for the provded data.
func BuildFingerprint(data []byte) (Fingerprint, error) {
	var fp Fingerprint

	hash := sha512.New384()
	if _, err := hash.Write([]byte(data)); err != nil {
		return fp, errors.Trace(err)
	}
	fp.raw = hash.Sum(nil)

	return fp, nil
}

// Raw returns the underlying hash for the fingerprint.
func (fp Fingerprint) Raw() []byte {
	raw := make([]byte, len(fp.raw))
	copy(raw, fp.raw)
	return raw
}

// Hex returns the hex string representing the fingerprint.
func (fp Fingerprint) Hex() string {
	return fmt.Sprintf("%x", fp.raw)
}

// Validate returns an error if the fingerprint is invalid.
func (fp Fingerprint) Validate() error {
	if len(fp.raw) < fingerprintSize {
		return errors.NewNotValid(nil, "invalid fingerprint (too small)")
	}
	if len(fp.raw) > fingerprintSize {
		return errors.NewNotValid(nil, "invalid fingerprint (too big)")
	}

	return nil
}
