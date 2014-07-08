// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import "os"

// The Bundle interface is implemented by any type that
// may be handled as a bundle.
type Bundle interface {
	Data() *BundleData
	ReadMe() string
}

// ReadBundle reads a Bundle from path, which can point to either a
// bundle archive or a bundle directory.
func ReadBundle(
	path string,
	verifyConstraints func(c string) error,
) (Bundle, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return ReadBundleDir(path, verifyConstraints)
	}
	return ReadBundleArchive(path, verifyConstraints)
}
