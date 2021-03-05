// Copyright 2021 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import "io"

// Manifest represents the recording of the building of the charm or bundle.
// The manifest file should represent the metadata.yaml, but a lot more
// information.
type Manifest struct {
	// TODO (stickupkid): Represent architectures in the future.
}

// ReadManifest reads in a Manifest from a charm's manifest.yaml.
// It is not validated at this point so that the caller can choose to override
// any validation.
func ReadManifest(r io.Reader) (*Manifest, error) {
	return &Manifest{}, nil
}
