// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import "fmt"

// Source represents the source of the charm.
type Source string

const (
	// Local represents a local charm.
	Local Source = "local"
	// CharmStore represents a charm from the now old charmstore.
	CharmStore Source = "charm-store"
	// Charmhub represents a charm from the new charmhub.
	Charmhub Source = "charmhub"
	// Unknown represents that we don't know where this charm came from. Either
	// the charm was migrated up from an older version of Juju or we didn't
	// have enough information when we set the charm.
	Unknown Source = "unknown"
)

// Originator represents the source of a charm.
type Originator interface {
	// Origin returns the source of the charm.
	Origin() *Origin
}

// Origin holds the information about where the charm originally came from.
type Origin struct {
	Source Source
}

// Validate the origin of a charm to ensure it's valid.
func (o *Origin) Validate() error {
	switch o.Source {
	case Local, CharmStore, Charmhub, Unknown:
	default:
		return fmt.Errorf("invalid source: %q", o.Source)
	}

	return nil
}
