// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"os"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("juju.charm")

// CharmMeta describes methods that inform charm operation.
type CharmMeta interface {
	Meta() *Meta
	Manifest() *Manifest
}

// The Charm interface is implemented by any type that
// may be handled as a charm.
type Charm interface {
	CharmMeta
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
		return nil, errors.Trace(err)
	}
	if info.IsDir() {
		charm, err = ReadCharmDir(path)
	} else {
		charm, err = ReadCharmArchive(path)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}

	return charm, errors.Trace(CheckMeta(charm))
}

// CheckMeta determines the version of the metadata used by this charm,
// then checks that it is valid as appropriate.
func CheckMeta(ch CharmMeta) error {
	manifest := ch.Manifest()

	format := FormatV2
	if manifest == nil || len(manifest.Bases) == 0 || len(ch.Meta().Series) > 0 {
		format = FormatV1
	}

	return ch.Meta().Check(format)
}

// SeriesForCharm takes a requested series and a list of series supported by a
// charm and returns the series which is relevant.
// If the requested series is empty, then the first supported series is used,
// otherwise the requested series is validated against the supported series.
func SeriesForCharm(requestedSeries string, supportedSeries []string) (string, error) {
	// Old charm with no supported series.
	if len(supportedSeries) == 0 {
		if requestedSeries == "" {
			return "", errMissingSeries
		}
		return requestedSeries, nil
	}
	// Use the charm default.
	if requestedSeries == "" {
		return supportedSeries[0], nil
	}
	for _, s := range supportedSeries {
		if s == requestedSeries {
			return requestedSeries, nil
		}
	}
	return "", &unsupportedSeriesError{requestedSeries, supportedSeries}
}

// errMissingSeries is used to denote that SeriesForCharm could not determine
// a series because a legacy charm did not declare any.
var errMissingSeries = fmt.Errorf("series not specified and charm does not define any")

// IsMissingSeriesError returns true if err is an errMissingSeries.
func IsMissingSeriesError(err error) bool {
	return err == errMissingSeries
}

// UnsupportedSeriesError represents an error indicating that the requested series
// is not supported by the charm.
type unsupportedSeriesError struct {
	requestedSeries string
	supportedSeries []string
}

func (e *unsupportedSeriesError) Error() string {
	return fmt.Sprintf(
		"series %q not supported by charm, supported series are: %s",
		e.requestedSeries, strings.Join(e.supportedSeries, ","),
	)
}

// NewUnsupportedSeriesError returns an error indicating that the requested series
// is not supported by a charm.
func NewUnsupportedSeriesError(requestedSeries string, supportedSeries []string) error {
	return &unsupportedSeriesError{requestedSeries, supportedSeries}
}

// IsUnsupportedSeriesError returns true if err is an UnsupportedSeriesError.
func IsUnsupportedSeriesError(err error) bool {
	_, ok := err.(*unsupportedSeriesError)
	return ok
}
