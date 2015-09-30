// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"os"
	"strings"

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

// SeriesToUse takes a specified series and a list of series supported by a
// charm and returns the series which is relevant.
// If the specified series is empty, the any default series defined by the
// charm is used. Any specified series is validated against those supported
// by the charm.
func SeriesToUse(series string, supportedSeries []string) (string, error) {
	// Old charm with no supported series.
	if len(supportedSeries) == 0 {
		return series, nil
	}
	// Use the charm default.
	if series == "" {
		return supportedSeries[0], nil
	}
	// Ensure series is supported.
	for _, s := range supportedSeries {
		if s == series {
			return series, nil
		}
	}
	return "", fmt.Errorf(
		"series %q not supported by charm, supported series are: %s",
		series, strings.Join(supportedSeries, ","),
	)
}
