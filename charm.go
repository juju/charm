// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"os"

	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("juju.charm")

// The Charm interface is implemented by any type that
// may be handled as a charm.
type Charm interface {
	Meta() *Meta
	Config() *Config
	Metrics() *Metrics
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
