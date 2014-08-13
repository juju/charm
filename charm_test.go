// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	stdtesting "testing"

	"gopkg.in/juju/charm.v3"
	charmtesting "gopkg.in/juju/charm.v3/testing"
	gc "launchpad.net/gocheck"
	goyaml "gopkg.in/yaml.v1"
)

func Test(t *stdtesting.T) {
	gc.TestingT(t)
}

type CharmSuite struct{}

var _ = gc.Suite(&CharmSuite{})

func (s *CharmSuite) TestReadCharm(c *gc.C) {
	bPath := charmtesting.Charms.CharmArchivePath(c.MkDir(), "dummy")
	ch, err := charm.ReadCharm(bPath)
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Meta().Name, gc.Equals, "dummy")
	dPath := charmtesting.Charms.CharmDirPath("dummy")
	ch, err = charm.ReadCharm(dPath)
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Meta().Name, gc.Equals, "dummy")
}

func (s *CharmSuite) TestReadCharmDirError(c *gc.C) {
	ch, err := charm.ReadCharm(c.MkDir())
	c.Assert(err, gc.NotNil)
	c.Assert(ch, gc.Equals, nil)
}

func (s *CharmSuite) TestReadCharmArchiveError(c *gc.C) {
	path := filepath.Join(c.MkDir(), "path")
	err := ioutil.WriteFile(path, []byte("foo"), 0644)
	c.Assert(err, gc.IsNil)
	ch, err := charm.ReadCharm(path)
	c.Assert(err, gc.NotNil)
	c.Assert(ch, gc.Equals, nil)
}

var inferRepoTests = []struct {
	url  string
	path string
}{
	{"cs:precise/wordpress", ""},
	{"local:oneiric/wordpress", "/some/path"},
}

func (s *CharmSuite) TestInferRepository(c *gc.C) {
	for i, t := range inferRepoTests {
		c.Logf("test %d", i)
		ref, err := charm.ParseReference(t.url)
		c.Assert(err, gc.IsNil)
		repo, err := charm.InferRepository(ref, "/some/path")
		c.Assert(err, gc.IsNil)
		switch repo := repo.(type) {
		case *charm.LocalRepository:
			c.Assert(repo.Path, gc.Equals, t.path)
		default:
			c.Assert(repo, gc.Equals, charm.Store)
		}
	}
	ref, err := charm.ParseReference("local:whatever")
	c.Assert(err, gc.IsNil)
	_, err = charm.InferRepository(ref, "")
	c.Assert(err, gc.ErrorMatches, "path to local repository not specified")
	ref.Schema = "foo"
	_, err = charm.InferRepository(ref, "")
	c.Assert(err, gc.ErrorMatches, "unknown schema for charm reference.*")
}

func checkDummy(c *gc.C, f charm.Charm, path string) {
	c.Assert(f.Revision(), gc.Equals, 1)
	c.Assert(f.Meta().Name, gc.Equals, "dummy")
	c.Assert(f.Config().Options["title"].Default, gc.Equals, "My Title")
	c.Assert(f.Actions(), gc.DeepEquals,
		&charm.Actions{
			map[string]charm.ActionSpec{
				"snapshot": charm.ActionSpec{
					Description: "Take a snapshot of the database.",
					Params: map[string]interface{}{
						"outfile": map[string]interface{}{
							"description": "The file to write out to.",
							"type":        "string",
							"default":     "foo.bz2",
						}}}}})
	switch f := f.(type) {
	case *charm.CharmArchive:
		c.Assert(f.Path, gc.Equals, path)
	case *charm.CharmDir:
		c.Assert(f.Path, gc.Equals, path)
	}
}

type YamlHacker map[interface{}]interface{}

func ReadYaml(r io.Reader) YamlHacker {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	m := make(map[interface{}]interface{})
	err = goyaml.Unmarshal(data, m)
	if err != nil {
		panic(err)
	}
	return YamlHacker(m)
}

func (yh YamlHacker) Reader() io.Reader {
	data, err := goyaml.Marshal(yh)
	if err != nil {
		panic(err)
	}
	return bytes.NewBuffer(data)
}
