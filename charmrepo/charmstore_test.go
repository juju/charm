// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charmstore.v4/csclient"

	"gopkg.in/juju/charm.v5-unstable/charmrepo"
)

type charmStoreSuite struct{}

var _ = gc.Suite(&charmStoreSuite{})

func (s *charmStoreSuite) TestURL(c *gc.C) {
	store := newStoreRepo(c, "https://1.2.3.4/charmstore").(*charmrepo.CharmStore)
	c.Assert(store.URL(), gc.Equals, "https://1.2.3.4/charmstore")
}

func (s *charmStoreSuite) TestDefaultURL(c *gc.C) {
	store := newStoreRepo(c, "").(*charmrepo.CharmStore)
	c.Assert(store.URL(), gc.Equals, csclient.ServerURL)
}

func (s *charmStoreSuite) TestCacheDir(c *gc.C) {
	cacheDir := c.MkDir()
	repo, err := charmrepo.NewCharmStore(charmrepo.NewCharmStoreParams{
		CacheDir: cacheDir,
	})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(charmrepo.CharmStoreCacheDir(repo), gc.Equals, cacheDir)
}

func (s *charmStoreSuite) TestCacheDirError(c *gc.C) {
	repo, err := charmrepo.NewCharmStore(charmrepo.NewCharmStoreParams{})
	c.Assert(err, gc.ErrorMatches, "charm cache directory path is empty")
	c.Assert(repo, gc.IsNil)
}

func (s *charmStoreSuite) TestTestMode(c *gc.C) {
	repo := newStoreRepo(c, "")

	// By default, test mode is disabled.
	c.Assert(charmrepo.CharmStoreTestMode(repo), jc.IsFalse)

	// Enable test mode.
	store := repo.(*charmrepo.CharmStore)
	repo = store.WithTestMode(true)
	c.Assert(charmrepo.CharmStoreTestMode(repo), jc.IsTrue)

	// Disable test mode again.
	repo = store.WithTestMode(false)
	c.Assert(charmrepo.CharmStoreTestMode(repo), jc.IsFalse)
}

// newStoreRepo creates and returns a charm store with the given URL.
// The cache directory is set to a temporary path.
func newStoreRepo(c *gc.C, url string) charmrepo.Interface {
	store, err := charmrepo.NewCharmStore(charmrepo.NewCharmStoreParams{
		URL:      url,
		CacheDir: c.MkDir(),
	})
	c.Assert(err, jc.ErrorIsNil)
	return store
}
