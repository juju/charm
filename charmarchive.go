// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/juju/utils/set"
	ziputil "github.com/juju/utils/zip"
)

// The CharmArchive type encapsulates access to data and operations
// on a charm archive.
type CharmArchive struct {
	Path     string // May be empty if CharmArchive wasn't read from a file
	meta     *Meta
	config   *Config
	actions  *Actions
	revision int
	r        io.ReaderAt
	size     int64
}

// Trick to ensure *CharmArchive implements the Charm interface.
var _ Charm = (*CharmArchive)(nil)

// ReadCharmArchive returns a CharmArchive for the charm in path.
func ReadCharmArchive(path string) (archive *CharmArchive, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return
	}
	b, err := readCharmArchive(f, fi.Size())
	if err != nil {
		return
	}
	b.Path = path
	return b, nil
}

// ReadCharmArchiveBytes returns a CharmArchive read from the given data.
// Make sure the archive fits in memory before using this.
func ReadCharmArchiveBytes(data []byte) (archive *CharmArchive, err error) {
	return readCharmArchive(readAtBytes(data), int64(len(data)))
}

func readCharmArchive(r io.ReaderAt, size int64) (archive *CharmArchive, err error) {
	b := &CharmArchive{r: r, size: size}
	zipr, err := zip.NewReader(r, size)
	if err != nil {
		return
	}
	reader, err := zipOpen(zipr, "metadata.yaml")
	if err != nil {
		return
	}
	b.meta, err = ReadMeta(reader)
	reader.Close()
	if err != nil {
		return
	}

	reader, err = zipOpen(zipr, "config.yaml")
	if _, ok := err.(*noCharmArchiveFile); ok {
		b.config = NewConfig()
	} else if err != nil {
		return nil, err
	} else {
		b.config, err = ReadConfig(reader)
		reader.Close()
		if err != nil {
			return nil, err
		}
	}

	reader, err = zipOpen(zipr, "actions.yaml")
	if _, ok := err.(*noCharmArchiveFile); ok {
		b.actions = NewActions()
	} else if err != nil {
		return nil, err
	} else {
		b.actions, err = ReadActionsYaml(reader)
		reader.Close()
		if err != nil {
			return nil, err
		}
	}

	reader, err = zipOpen(zipr, "revision")
	if err != nil {
		if _, ok := err.(*noCharmArchiveFile); !ok {
			return
		}
		b.revision = b.meta.OldRevision
	} else {
		_, err = fmt.Fscan(reader, &b.revision)
		if err != nil {
			return nil, errors.New("invalid revision file")
		}
	}

	return b, nil
}

func zipOpen(zipr *zip.Reader, path string) (rc io.ReadCloser, err error) {
	for _, fh := range zipr.File {
		if fh.Name == path {
			return fh.Open()
		}
	}
	return nil, &noCharmArchiveFile{path}
}

type noCharmArchiveFile struct {
	path string
}

func (err noCharmArchiveFile) Error() string {
	return fmt.Sprintf("archive file not found: %s", err.path)
}

// Revision returns the revision number for the charm
// expanded in dir.
func (a *CharmArchive) Revision() int {
	return a.revision
}

// SetRevision changes the charm revision number. This affects the
// revision reported by Revision and the revision of the charm
// directory created by ExpandTo.
func (a *CharmArchive) SetRevision(revision int) {
	a.revision = revision
}

// Meta returns the Meta representing the metadata.yaml file from archive.
func (a *CharmArchive) Meta() *Meta {
	return a.meta
}

// Config returns the Config representing the config.yaml file
// for the charm archive.
func (a *CharmArchive) Config() *Config {
	return a.config
}

// Actions returns the Actions map for the actions.yaml file for the charm
// archive.
func (a *CharmArchive) Actions() *Actions {
	return a.actions
}

type zipReadCloser struct {
	io.Closer
	*zip.Reader
}

// zipOpen returns a zipReadCloser.
func (a *CharmArchive) zipOpen() (*zipReadCloser, error) {
	// If we don't have a Path, try to use the original ReaderAt.
	if a.Path == "" {
		r, err := zip.NewReader(a.r, a.size)
		if err != nil {
			return nil, err
		}
		return &zipReadCloser{Closer: ioutil.NopCloser(nil), Reader: r}, nil
	}
	f, err := os.Open(a.Path)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	r, err := zip.NewReader(f, fi.Size())
	if err != nil {
		f.Close()
		return nil, err
	}
	return &zipReadCloser{Closer: f, Reader: r}, nil
}

// Manifest returns a set of the charm's contents.
func (a *CharmArchive) Manifest() (set.Strings, error) {
	zipr, err := a.zipOpen()
	if err != nil {
		return set.NewStrings(), err
	}
	defer zipr.Close()
	paths, err := ziputil.Find(zipr.Reader, "*")
	if err != nil {
		return set.NewStrings(), err
	}
	manifest := set.NewStrings(paths...)
	// We always write out a revision file, even if there isn't one in the
	// archive; and we always strip ".", because that's sometimes not present.
	manifest.Add("revision")
	manifest.Remove(".")
	return manifest, nil
}

// ExpandTo expands the charm archive into dir, creating it if necessary.
// If any errors occur during the expansion procedure, the process will
// abort.
func (a *CharmArchive) ExpandTo(dir string) (err error) {
	zipr, err := a.zipOpen()
	if err != nil {
		return err
	}
	defer zipr.Close()
	if err := ziputil.ExtractAll(zipr.Reader, dir); err != nil {
		return err
	}
	hooksDir := filepath.Join(dir, "hooks")
	fixHook := fixHookFunc(hooksDir, a.meta.Hooks())
	if err := filepath.Walk(hooksDir, fixHook); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	revFile, err := os.Create(filepath.Join(dir, "revision"))
	if err != nil {
		return err
	}
	_, err = revFile.Write([]byte(strconv.Itoa(a.revision)))
	revFile.Close()
	return err
}

// fixHookFunc returns a WalkFunc that makes sure hooks are owner-executable.
func fixHookFunc(hooksDir string, hookNames map[string]bool) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		mode := info.Mode()
		if path != hooksDir && mode.IsDir() {
			return filepath.SkipDir
		}
		if name := filepath.Base(path); hookNames[name] {
			if mode&0100 == 0 {
				return os.Chmod(path, mode|0100)
			}
		}
		return nil
	}
}

// FWIW, being able to do this is awesome.
type readAtBytes []byte

func (b readAtBytes) ReadAt(out []byte, off int64) (n int, err error) {
	return copy(out, b[off:]), nil
}
