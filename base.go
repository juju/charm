// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"strings"

	"github.com/juju/collections/set"
	"github.com/juju/errors"
	"github.com/juju/os/v2"
)

// Base represents an OS/Channel.
// Bases can also be converted to and from a series string.
type Base struct {
	Name    string  `json:"name,omitempty"`
	Channel Channel `json:"channel,omitempty"`
}

// Validate returns with no error when the Base is valid.
func (s Base) Validate() error {
	if s.Name == "" {
		return errors.NotValidf("name must be specified")
	}

	if !validOSForBase.Contains(s.Name) {
		return errors.NotValidf("os %q", s.Name)
	}
	if s.Channel.Empty() {
		return errors.NotValidf("channel")
	}

	return nil
}

// String representation of the Base.
func (s Base) String() string {
	str := strings.ToLower(s.Name)
	if !s.Channel.Empty() {
		str += "/" + s.Channel.String()
	}
	return str
}

// ParseBaseFromString parses a base as series string
// in the form "os/track/risk/branch"
func ParseBaseFromString(s string) (Base, error) {
	var err error
	base := Base{}

	// Split the first forward-slash to get name and channel.
	// E.g. "os/track/risk/branch" => ["os", "track/risk/branch"]
	segments := strings.SplitN(s, "/", 2)
	base.Name = strings.ToLower(segments[0])
	channelName := ""
	if len(segments) == 2 {
		channelName = segments[1]
	}

	if channelName != "" {
		base.Channel, err = ParseChannelNormalize(channelName)
		if err != nil {
			return Base{}, errors.Annotatef(err, "malformed channel in base string %q", s)
		}
	}

	err = base.Validate()
	if err != nil {
		return Base{}, errors.Annotatef(err, "invalid base string %q", s)
	}
	return base, nil
}

// validOSForBase is a string set of valid OS names for a base.
var validOSForBase = set.NewStrings(
	strings.ToLower(os.Ubuntu.String()),
	strings.ToLower(os.CentOS.String()),
	strings.ToLower(os.Windows.String()),
	strings.ToLower(os.OSX.String()),
	strings.ToLower(os.OpenSUSE.String()),
	strings.ToLower(os.GenericLinux.String()),
)
