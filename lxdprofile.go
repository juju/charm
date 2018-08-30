// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/utils/set"
	lxdapi "github.com/lxc/lxd/shared/api"
	"gopkg.in/yaml.v2"
)

type LXDProfile struct {
	lxdapi.Profile
}

func NewLXDProfile() *LXDProfile {
	return &LXDProfile{}
}

// ReadLXDProfile reads in a LXDProfile from a charm's lxd-profile.yaml.
// It is not validated at this point so that the caller can choose to override
// any validation.
func ReadLXDProfile(r io.Reader) (*LXDProfile, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var profile LXDProfile
	if err := yaml.Unmarshal(data, &profile.Profile); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshall lxd-profile.yaml")
	}
	return &profile, nil
}

// HasName returns true if the lxd profile name is set
func (profile *LXDProfile) HasName() bool {
	return profile.Name != ""
}

// ValidateConfigDevices validates the Config and Devices properties of the LXDProfile.
// WhiteList devices: unix-char, unix-block, gpu, usb.
// BlackList config: boot*, limits* and migration*.
func (profile *LXDProfile) ValidateConfigDevices() error {
	if len(profile.Devices) < 1 && len(profile.Config) < 1 {
		return fmt.Errorf("invalid lxd-profile.yaml: does not contain devices nor config")
	}
	for _, val := range profile.Devices {
		goodDevs := set.NewStrings("unix-char", "unix-block", "gpu", "usb")
		if devType, ok := val["type"]; ok {
			if !goodDevs.Contains(devType) {
				return fmt.Errorf("invalid lxd-profile.yaml: contains device type %q", devType)
			}
		}
	}
	for key, _ := range profile.Config {
		if strings.HasPrefix(key, "boot") ||
			strings.HasPrefix(key, "limits") ||
			strings.HasPrefix(key, "migration") {
			return fmt.Errorf("invalid lxd-profile.yaml: contains config value %q", key)
		}
	}
	return nil
}
