// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/errgo.v1"

	"gopkg.in/juju/charm.v5-unstable"
)

// LocalRepository represents a local directory containing subdirectories
// named after an Ubuntu series, each of which contains charms targeted for
// that series. For example:
//
//   /path/to/repository/oneiric/mongodb/
//   /path/to/repository/precise/mongodb.charm
//   /path/to/repository/precise/wordpress/
type LocalRepository struct {
	Path          string
	defaultSeries string
}

var _ Interface = (*LocalRepository)(nil)

// NewLocalRepository creates and return a new local Juju repository pointing
// to the given local path.
func NewLocalRepository(path string) (Interface, error) {
	if path == "" {
		return nil, errgo.New("path to local repository not specified")
	}
	return &LocalRepository{
		Path: path,
	}, nil
}

// WithDefaultSeries returns a repository Interface with the default series
// set.
func (r *LocalRepository) WithDefaultSeries(defaultSeries string) Interface {
	localRepo := *r
	localRepo.defaultSeries = defaultSeries
	return &localRepo
}

// Resolve canonicalizes charm URLs, resolving references and implied series.
func (r *LocalRepository) Resolve(ref *charm.Reference) (*charm.URL, error) {
	return ref.URL(r.defaultSeries)
}

// Latest returns the latest revision of the charm referenced by curl, regardless
// of the revision set on curl itself.
func (r *LocalRepository) Latest(curls ...*charm.URL) ([]CharmRevision, error) {
	result := make([]CharmRevision, len(curls))
	for i, curl := range curls {
		ch, err := r.Get(curl.WithRevision(-1))
		if err == nil {
			result[i].Revision = ch.Revision()
		} else {
			result[i].Err = err
		}
	}
	return result, nil
}

func mightBeCharm(info os.FileInfo) bool {
	if info.IsDir() {
		return !strings.HasPrefix(info.Name(), ".")
	}
	return strings.HasSuffix(info.Name(), ".charm")
}

// Get returns a charm matching curl, if one exists. If curl has a revision of
// -1, it returns the latest charm that matches curl. If multiple candidates
// satisfy the foregoing, the first one encountered will be returned.
func (r *LocalRepository) Get(curl *charm.URL) (charm.Charm, error) {
	if curl.Schema != "local" {
		return nil, fmt.Errorf("local repository got URL with non-local schema: %q", curl)
	}
	info, err := os.Stat(r.Path)
	if err != nil {
		if os.IsNotExist(err) {
			err = repoNotFound(r.Path)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, repoNotFound(r.Path)
	}
	path := filepath.Join(r.Path, curl.Series)
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, charmNotFound(curl, r.Path)
	}
	var latest charm.Charm
	for _, info := range infos {
		chPath := filepath.Join(path, info.Name())
		if info.Mode()&os.ModeSymlink != 0 {
			var err error
			if info, err = os.Stat(chPath); err != nil {
				return nil, err
			}
		}
		if !mightBeCharm(info) {
			continue
		}
		if ch, err := charm.ReadCharm(chPath); err != nil {
			logger.Warningf("failed to load charm at %q: %s", chPath, err)
		} else if ch.Meta().Name == curl.Name {
			if ch.Revision() == curl.Revision {
				return ch, nil
			}
			if latest == nil || ch.Revision() > latest.Revision() {
				latest = ch
			}
		}
	}
	if curl.Revision == -1 && latest != nil {
		return latest, nil
	}
	return nil, charmNotFound(curl, r.Path)
}
