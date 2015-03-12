// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	gitjujutesting "github.com/juju/testing"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v5-unstable"
	"gopkg.in/juju/charm.v5-unstable/charmrepo"
)

type LocalRepoSuite struct {
	gitjujutesting.FakeHomeSuite
	repo       *charmrepo.LocalRepository
	seriesPath string
}

var _ = gc.Suite(&LocalRepoSuite{})

func (s *LocalRepoSuite) SetUpTest(c *gc.C) {
	s.FakeHomeSuite.SetUpTest(c)
	root := c.MkDir()
	s.repo = &charmrepo.LocalRepository{Path: root}
	s.seriesPath = filepath.Join(root, "quantal")
	c.Assert(os.Mkdir(s.seriesPath, 0777), gc.IsNil)
}

func (s *LocalRepoSuite) addCharmArchive(name string) string {
	return TestCharms.CharmArchivePath(s.seriesPath, name)
}

func (s *LocalRepoSuite) addDir(name string) string {
	return TestCharms.ClonedDirPath(s.seriesPath, name)
}

func (s *LocalRepoSuite) checkNotFoundErr(c *gc.C, err error, charmURL *charm.URL) {
	expect := `charm not found in "` + s.repo.Path + `": ` + charmURL.String()
	c.Check(err, gc.ErrorMatches, expect)
}

func (s *LocalRepoSuite) TestMissingCharm(c *gc.C) {
	for i, str := range []string{
		"local:quantal/zebra", "local:badseries/zebra",
	} {
		c.Logf("test %d: %s", i, str)
		charmURL := charm.MustParseURL(str)
		_, err := charmrepo.Latest(s.repo, charmURL)
		s.checkNotFoundErr(c, err, charmURL)
		_, err = s.repo.Get(charmURL)
		s.checkNotFoundErr(c, err, charmURL)
	}
}

func (s *LocalRepoSuite) TestMissingRepo(c *gc.C) {
	c.Assert(os.RemoveAll(s.repo.Path), gc.IsNil)
	_, err := charmrepo.Latest(s.repo, charm.MustParseURL("local:quantal/zebra"))
	c.Assert(err, gc.ErrorMatches, `no repository found at ".*"`)
	_, err = s.repo.Get(charm.MustParseURL("local:quantal/zebra"))
	c.Assert(err, gc.ErrorMatches, `no repository found at ".*"`)
	c.Assert(ioutil.WriteFile(s.repo.Path, nil, 0666), gc.IsNil)
	_, err = charmrepo.Latest(s.repo, charm.MustParseURL("local:quantal/zebra"))
	c.Assert(err, gc.ErrorMatches, `no repository found at ".*"`)
	_, err = s.repo.Get(charm.MustParseURL("local:quantal/zebra"))
	c.Assert(err, gc.ErrorMatches, `no repository found at ".*"`)
}

func (s *LocalRepoSuite) TestMultipleVersions(c *gc.C) {
	charmURL := charm.MustParseURL("local:quantal/upgrade")
	s.addDir("upgrade1")
	rev, err := charmrepo.Latest(s.repo, charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(rev, gc.Equals, 1)
	ch, err := s.repo.Get(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Revision(), gc.Equals, 1)

	s.addDir("upgrade2")
	rev, err = charmrepo.Latest(s.repo, charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(rev, gc.Equals, 2)
	ch, err = s.repo.Get(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Revision(), gc.Equals, 2)

	revCharmURL := charmURL.WithRevision(1)
	rev, err = charmrepo.Latest(s.repo, revCharmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(rev, gc.Equals, 2)
	ch, err = s.repo.Get(revCharmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Revision(), gc.Equals, 1)

	badRevCharmURL := charmURL.WithRevision(33)
	rev, err = charmrepo.Latest(s.repo, badRevCharmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(rev, gc.Equals, 2)
	_, err = s.repo.Get(badRevCharmURL)
	s.checkNotFoundErr(c, err, badRevCharmURL)
}

func (s *LocalRepoSuite) TestCharmArchive(c *gc.C) {
	charmURL := charm.MustParseURL("local:quantal/dummy")
	s.addCharmArchive("dummy")

	rev, err := charmrepo.Latest(s.repo, charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(rev, gc.Equals, 1)
	ch, err := s.repo.Get(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Revision(), gc.Equals, 1)
}

func (s *LocalRepoSuite) TestLogsErrors(c *gc.C) {
	err := ioutil.WriteFile(filepath.Join(s.seriesPath, "blah.charm"), nil, 0666)
	c.Assert(err, gc.IsNil)
	err = os.Mkdir(filepath.Join(s.seriesPath, "blah"), 0666)
	c.Assert(err, gc.IsNil)
	samplePath := s.addDir("upgrade2")
	gibberish := []byte("don't parse me by")
	err = ioutil.WriteFile(filepath.Join(samplePath, "metadata.yaml"), gibberish, 0666)
	c.Assert(err, gc.IsNil)

	charmURL := charm.MustParseURL("local:quantal/dummy")
	s.addDir("dummy")
	ch, err := s.repo.Get(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Revision(), gc.Equals, 1)
	c.Assert(c.GetTestLog(), gc.Matches, `
.* WARNING juju.charm.charmrepo failed to load charm at ".*/quantal/blah": .*
.* WARNING juju.charm.charmrepo failed to load charm at ".*/quantal/blah.charm": .*
.* WARNING juju.charm.charmrepo failed to load charm at ".*/quantal/upgrade2": .*
`[1:])
}

func renameSibling(c *gc.C, path, name string) {
	c.Assert(os.Rename(path, filepath.Join(filepath.Dir(path), name)), gc.IsNil)
}

func (s *LocalRepoSuite) TestIgnoresUnpromisingNames(c *gc.C) {
	err := ioutil.WriteFile(filepath.Join(s.seriesPath, "blah.notacharm"), nil, 0666)
	c.Assert(err, gc.IsNil)
	err = os.Mkdir(filepath.Join(s.seriesPath, ".blah"), 0666)
	c.Assert(err, gc.IsNil)
	renameSibling(c, s.addDir("dummy"), ".dummy")
	renameSibling(c, s.addCharmArchive("dummy"), "dummy.notacharm")
	charmURL := charm.MustParseURL("local:quantal/dummy")

	_, err = s.repo.Get(charmURL)
	s.checkNotFoundErr(c, err, charmURL)
	_, err = charmrepo.Latest(s.repo, charmURL)
	s.checkNotFoundErr(c, err, charmURL)
	c.Assert(c.GetTestLog(), gc.Equals, "")
}

func (s *LocalRepoSuite) TestFindsSymlinks(c *gc.C) {
	realPath := TestCharms.ClonedDirPath(c.MkDir(), "dummy")
	linkPath := filepath.Join(s.seriesPath, "dummy")
	err := os.Symlink(realPath, linkPath)
	c.Assert(err, gc.IsNil)
	ch, err := s.repo.Get(charm.MustParseURL("local:quantal/dummy"))
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Revision(), gc.Equals, 1)
	c.Assert(ch.Meta().Name, gc.Equals, "dummy")
	c.Assert(ch.Config().Options["title"].Default, gc.Equals, "My Title")
	c.Assert(ch.(*charm.CharmDir).Path, gc.Equals, linkPath)
}
