// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"os"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/version"
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

// SeriesForCharm takes a requested series and a list of series supported by a
// charm and returns the series which is relevant.
// If the requested series is empty, then the first supported series is used,
// otherwise the requested series is validated against the supported series.
func SeriesForCharm(requestedSeries string, supportedSeries []string) (string, error) {
	// Old charm with no supported series.
	if len(supportedSeries) == 0 {
		if requestedSeries == "" {
			return "", missingSeriesError
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

// missingSeriesError is used to denote that SeriesForCharm could not determine
// a series because a legacy charm did not declare any.
var missingSeriesError = fmt.Errorf("series not specified and charm does not define any")

// IsMissingSeriesError returns true if err is an missingSeriesError.
func IsMissingSeriesError(err error) bool {
	return err == missingSeriesError
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

// isMinVersionError reports whether the given error indicates a minmum version
// too high for this version of juju.
func IsMinVersionError(err error) bool {
	_, ok := err.(minJujuVersionErr)
	return ok
}

// CheckMinVersion reports an error that reports true from IsMinVersionErr if
// the given charm has a minimum juju version specified that is higher than the
// given juju version number.
func CheckMinVersion(ch Charm, jujuVersion version.Number) error {
	minver := ch.Meta().MinJujuVersion
	if minver != nil && minver.Compare(jujuVersion) > 0 {
		err := errors.NewErr("charm's min version (%s) is higher than this juju environment's version (%s)",
			minver, jujuVersion)
		return minJujuVersionErr{&err}
	}
	return nil
}

type minJujuVersionErr struct {
	*errors.Err
}
