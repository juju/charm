// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

type BundleArchive struct {
	Path string
}

func ReadBundleArchive(
	path string,
	verifyConstraints func(c string) error,
) (dir *BundleArchive, err error) {
	panic("unimplemented")
}

func (dir *BundleArchive) Data() *BundleData {
	panic("unimplemented")
}

func (dir *BundleArchive) ReadMe() string {
	panic("unimplemented")
}
