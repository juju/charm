// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo_test

import (
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v5-unstable"
	"gopkg.in/juju/charm.v5-unstable/charmrepo"
	charmtesting "gopkg.in/juju/charm.v5-unstable/testing"
)

var TestCharms = charmtesting.NewRepo("../internal/test-charm-repo", "quantal")

type inferRepoSuite struct{}

var _ = gc.Suite(&inferRepoSuite{})

var inferRepoTests = []struct {
	url  string
	path string
}{
	{"cs:precise/wordpress", ""},
	{"local:oneiric/wordpress", "/some/path"},
}

func (s *inferRepoSuite) TestInferRepository(c *gc.C) {
	for i, t := range inferRepoTests {
		c.Logf("test %d", i)
		ref, err := charm.ParseReference(t.url)
		c.Assert(err, gc.IsNil)
		repo, err := charmrepo.InferRepository(ref, "/some/path")
		c.Assert(err, gc.IsNil)
		switch repo := repo.(type) {
		case *charmrepo.LocalRepository:
			c.Assert(repo.Path, gc.Equals, t.path)
		default:
			c.Assert(repo, gc.Equals, charmrepo.Store)
		}
	}
	ref, err := charm.ParseReference("local:whatever")
	c.Assert(err, gc.IsNil)
	_, err = charmrepo.InferRepository(ref, "")
	c.Assert(err, gc.ErrorMatches, "path to local repository not specified")
	ref.Schema = "foo"
	_, err = charmrepo.InferRepository(ref, "")
	c.Assert(err, gc.ErrorMatches, "unknown schema for charm reference.*")
}
