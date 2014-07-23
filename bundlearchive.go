// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"io/ioutil"

	ziputil "github.com/juju/utils/zip"
)

type BundleArchive struct {
	zopen zipOpener

	Path   string
	data   *BundleData
	readMe string
}

// ReadBundleArchive reads a bundle archive from the given file
// path, using verifyConstraints to verify any constraints found
// in the bundle.yaml file.
func ReadBundleArchive(
	path string,
	verifyConstraints func(c string) error,
) (*BundleArchive, error) {
	a, err := readBundleArchive(newZipOpenerFromPath(path), verifyConstraints)
	if err != nil {
		return nil, err
	}
	a.Path = path
	return a, nil
}

// ReadBundleArchiveBytes reads a bundle archive from the given byte
// slice, using verifyConstraints to verify any constraints found in the
// bundle.yaml file.
func ReadBundleArchiveBytes(
	data []byte,
	verifyConstraints func(c string) error,
) (*BundleArchive, error) {
	return readBundleArchive(newZipOpenerFromBytes(data), verifyConstraints)
}

func readBundleArchive(zopen zipOpener, verifyConstraints func(c string) error) (*BundleArchive, error) {
	a := &BundleArchive{
		zopen: zopen,
	}
	zipr, err := zopen.openZip()
	if err != nil {
		return nil, err
	}
	defer zipr.Close()
	reader, err := zipOpenFile(zipr, "bundle.yaml")
	if err != nil {
		return nil, err
	}
	a.data, err = ReadBundleData(reader)
	reader.Close()
	if err != nil {
		return nil, err
	}
	reader, err = zipOpenFile(zipr, "README.md")
	if err != nil {
		return nil, err
	}
	readMe, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	a.readMe = string(readMe)
	return a, nil
}

// Data implements Bundle.Data.
func (a *BundleArchive) Data() *BundleData {
	return a.data
}

// ReadMe implements Bundle.ReadMe.
func (a *BundleArchive) ReadMe() string {
	return a.readMe
}

// ExpandTo expands the bundle archive into dir, creating it if necessary.
// If any errors occur during the expansion procedure, the process will
// abort.
func (a *BundleArchive) ExpandTo(dir string) error {
	zipr, err := a.zopen.openZip()
	if err != nil {
		return err
	}
	defer zipr.Close()
	return ziputil.ExtractAll(zipr.Reader, dir)
}
