// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charmstore.v4/csclient"

	"gopkg.in/juju/charm.v5-unstable"
	"gopkg.in/juju/charm.v5-unstable/charmrepo"
	charmtesting "gopkg.in/juju/charm.v5-unstable/testing"
)

var TestCharms = charmtesting.NewRepo("../internal/test-charm-repo", "quantal")

type inferRepoSuite struct{}

var _ = gc.Suite(&inferRepoSuite{})

var inferRepositoryTests = []struct {
	url              string
	charmStoreParams charmrepo.NewCharmStoreParams
	localRepoPath    string
	err              string
}{{
	url: "cs:trusty/django",
	err: "charm cache directory path is empty",
}, {
	url: "local:precise/wordpress",
	err: "path to local repository not specified",
}, {
	url: "cs:trusty/wordpress-42",
	charmStoreParams: charmrepo.NewCharmStoreParams{
		CacheDir: "/tmp/cache-dir",
	},
}, {
	url:           "local:precise/haproxy-47",
	localRepoPath: "/tmp/repo-path",
}}

func (s *inferRepoSuite) TestInferRepository(c *gc.C) {
	for i, test := range inferRepositoryTests {
		c.Logf("test %d: %s", i, test.url)
		ref := charm.MustParseReference(test.url)
		repo, err := charmrepo.InferRepository(
			ref, test.charmStoreParams, test.localRepoPath)
		if test.err != "" {
			c.Assert(err, gc.ErrorMatches, test.err)
			c.Assert(repo, gc.IsNil)
			continue
		}
		c.Assert(err, jc.ErrorIsNil)
		switch store := repo.(type) {
		case *charmrepo.LocalRepository:
			c.Assert(store.Path, gc.Equals, test.localRepoPath)
		case *charmrepo.CharmStore:
			c.Assert(store.URL(), gc.Equals, csclient.ServerURL)
		default:
			c.Fatal("unknown repository type")
		}
	}
}
