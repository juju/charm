// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package resource

import (
	"crypto/sha512"
	"encoding/hex"

	"github.com/juju/errors"
)

const fingerprintSize = 48 // 384 / 8

// Fingerprint represents the unique fingerprint value of a resource's data.
type Fingerprint struct {
	raw string
}

// NewFingerprint returns wraps the provided raw fingerprint.
func NewFingerprint(raw []byte) (Fingerprint, error) {
	fp := Fingerprint{
		raw: string(raw),
	}
	if err := fp.validate(); err != nil {
		return Fingerprint{}, errors.Trace(err)
	}
	return fp, nil
}

// GenerateFingerprint returns the fingerprint for the provided data.
func GenerateFingerprint(data []byte) (Fingerprint, error) {
	var fp Fingerprint

	hash := sha512.New384()
	if _, err := hash.Write([]byte(data)); err != nil {
		return fp, errors.Trace(err)
	}
	fp.raw = string(hash.Sum(nil))

	return fp, nil
}

// String returns the hex string representation of the fingerprint.
func (fp Fingerprint) String() string {
	return hex.EncodeToString([]byte(fp.raw))
}

// Bytes returns the raw bytes of the fingerprint.
func (fp Fingerprint) Bytes() []byte {
	return []byte(fp.raw)
}

// Validate returns an error if the fingerprint is invalid.
func (fp Fingerprint) Validate() error {
	if fp.raw == "" {
		return errors.NotValidf("zero-value fingerprint")
	}
	return nil
}

func (fp Fingerprint) validate() error {
	if len(fp.raw) < fingerprintSize {
		return errors.NewNotValid(nil, "invalid fingerprint (too small)")
	}
	if len(fp.raw) > fingerprintSize {
		return errors.NewNotValid(nil, "invalid fingerprint (too big)")
	}

	return nil
}
