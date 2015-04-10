// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	stdtesting "testing"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/yaml.v1"

	"gopkg.in/juju/charm.v5"
	charmtesting "gopkg.in/juju/charm.v5/testing"
)

func Test(t *stdtesting.T) {
	gc.TestingT(t)
}

var TestCharms = charmtesting.NewRepo("internal/test-charm-repo", "quantal")

type CharmSuite struct{}

var _ = gc.Suite(&CharmSuite{})

func (s *CharmSuite) TestReadCharm(c *gc.C) {
	bPath := TestCharms.CharmArchivePath(c.MkDir(), "dummy")
	ch, err := charm.ReadCharm(bPath)
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Meta().Name, gc.Equals, "dummy")
	dPath := TestCharms.CharmDirPath("dummy")
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

func checkDummy(c *gc.C, f charm.Charm, path string) {
	c.Assert(f.Revision(), gc.Equals, 1)
	c.Assert(f.Meta().Name, gc.Equals, "dummy")
	c.Assert(f.Config().Options["title"].Default, gc.Equals, "My Title")
	c.Assert(f.Actions(), jc.DeepEquals,
		&charm.Actions{
			map[string]charm.ActionSpec{
				"snapshot": charm.ActionSpec{
					Description: "Take a snapshot of the database.",
					Params: map[string]interface{}{
						"type":        "object",
						"description": "Take a snapshot of the database.",
						"title":       "snapshot",
						"properties": map[string]interface{}{
							"outfile": map[string]interface{}{
								"description": "The file to write out to.",
								"type":        "string",
								"default":     "foo.bz2",
							}}}}}})
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
	err = yaml.Unmarshal(data, m)
	if err != nil {
		panic(err)
	}
	return YamlHacker(m)
}

func (yh YamlHacker) Reader() io.Reader {
	data, err := yaml.Marshal(yh)
	if err != nil {
		panic(err)
	}
	return bytes.NewBuffer(data)
}
