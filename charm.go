// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"errors"
	"fmt"
	"os"

	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("juju.charm")

// The Charm interface is implemented by any type that
// may be handled as a charm.
type Charm interface {
	Meta() *Meta
	Config() *Config
	Actions() *Actions
	Revision() int
}

// ReadCharm reads a Charm from path, which can point to either a charm archive or a
// charm directory.
func ReadCharm(path string) (charm Charm, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		charm, err = ReadCharmDir(path)
	} else {
		charm, err = ReadCharmArchive(path)
	}
	if err != nil {
		return nil, err
	}
	return charm, nil
}

// InferRepository returns a charm repository inferred from the provided charm
// or bundle reference. Local references will use the provided path.
func InferRepository(ref *Reference, localRepoPath string) (repo Repository, err error) {
	switch ref.Schema {
	case "cs":
		repo = Store
	case "local":
		if localRepoPath == "" {
			return nil, errors.New("path to local repository not specified")
		}
		repo = &LocalRepository{Path: localRepoPath}
	default:
		// TODO fix this error message to reference bundles too?
		return nil, fmt.Errorf("unknown schema for charm reference %q", ref)
	}
	return
}
