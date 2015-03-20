// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo

// CharmStoreCacheDir returns the charm cache path of the given charm store.
func CharmStoreCacheDir(r Interface) string {
	return r.(*CharmStore).cacheDir
}
