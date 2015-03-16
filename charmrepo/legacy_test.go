// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	gitjujutesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v5-unstable"
	"gopkg.in/juju/charm.v5-unstable/charmrepo"
	charmtesting "gopkg.in/juju/charm.v5-unstable/testing"
)

type legacyCharmStoreSuite struct {
	gitjujutesting.FakeHomeSuite
	server *charmtesting.MockStore
	store  *charmrepo.LegacyCharmStore
}

var _ = gc.Suite(&legacyCharmStoreSuite{})

func (s *legacyCharmStoreSuite) SetUpSuite(c *gc.C) {
	s.FakeHomeSuite.SetUpSuite(c)
	s.server = charmtesting.NewMockStore(c, TestCharms, map[string]int{
		"cs:series/good":   23,
		"cs:series/unwise": 23,
		"cs:series/better": 24,
		"cs:series/best":   25,
	})
}

func (s *legacyCharmStoreSuite) SetUpTest(c *gc.C) {
	s.FakeHomeSuite.SetUpTest(c)
	s.PatchValue(&charmrepo.CacheDir, c.MkDir())
	s.store = newLegacyStore(s.server.Address())
	s.server.Downloads = nil
	s.server.Authorizations = nil
	s.server.Metadata = nil
	s.server.DownloadsNoStats = nil
	s.server.InfoRequestCount = 0
	s.server.InfoRequestCountNoStats = 0
}

func (s *legacyCharmStoreSuite) TearDownSuite(c *gc.C) {
	s.server.Close()
	s.FakeHomeSuite.TearDownSuite(c)
}

func (s *legacyCharmStoreSuite) TestMissing(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/missing")
	expect := `charm not found: cs:series/missing`
	_, err := charmrepo.Latest(s.store, charmURL)
	c.Assert(err, gc.ErrorMatches, expect)
	_, err = s.store.Get(charmURL)
	c.Assert(err, gc.ErrorMatches, expect)
}

func (s *legacyCharmStoreSuite) TestError(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/borken")
	expect := `charm info errors for "cs:series/borken": badness`
	_, err := charmrepo.Latest(s.store, charmURL)
	c.Assert(err, gc.ErrorMatches, expect)
	_, err = s.store.Get(charmURL)
	c.Assert(err, gc.ErrorMatches, expect)
}

func (s *legacyCharmStoreSuite) TestWarning(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/unwise")
	expect := `.* WARNING juju.charm.charmrepo charm store reports for "cs:series/unwise": foolishness` + "\n"
	r, err := charmrepo.Latest(s.store, charmURL)
	c.Assert(r, gc.Equals, 23)
	c.Assert(err, gc.IsNil)
	c.Assert(c.GetTestLog(), gc.Matches, expect)
	ch, err := s.store.Get(charmURL)
	c.Assert(ch, gc.NotNil)
	c.Assert(err, gc.IsNil)
	c.Assert(c.GetTestLog(), gc.Matches, expect+expect)
}

func (s *legacyCharmStoreSuite) TestLatest(c *gc.C) {
	urls := []*charm.URL{
		charm.MustParseURL("cs:series/good"),
		charm.MustParseURL("cs:series/good-2"),
		charm.MustParseURL("cs:series/good-99"),
	}
	revInfo, err := s.store.Latest(urls...)
	c.Assert(err, gc.IsNil)
	c.Assert(revInfo, jc.DeepEquals, []charmrepo.CharmRevision{
		{23, "843f8bba130a9705249f038202fab24e5151e3a2f7b6626f4508a5725739a5b5", nil},
		{23, "843f8bba130a9705249f038202fab24e5151e3a2f7b6626f4508a5725739a5b5", nil},
		{23, "843f8bba130a9705249f038202fab24e5151e3a2f7b6626f4508a5725739a5b5", nil},
	})
}

func (s *legacyCharmStoreSuite) assertCached(c *gc.C, charmURL *charm.URL) {
	s.server.Downloads = nil
	ch, err := s.store.Get(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch, gc.NotNil)
	c.Assert(s.server.Downloads, gc.IsNil)
}

func (s *legacyCharmStoreSuite) TestGetCacheImplicitRevision(c *gc.C) {
	base := "cs:series/good"
	charmURL := charm.MustParseURL(base)
	revCharmURL := charm.MustParseURL(base + "-23")
	ch, err := s.store.Get(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch, gc.NotNil)
	c.Assert(s.server.Downloads, jc.DeepEquals, []*charm.URL{revCharmURL})
	s.assertCached(c, charmURL)
	s.assertCached(c, revCharmURL)
}

func (s *legacyCharmStoreSuite) TestGetCacheExplicitRevision(c *gc.C) {
	base := "cs:series/good-12"
	charmURL := charm.MustParseURL(base)
	ch, err := s.store.Get(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch, gc.NotNil)
	c.Assert(s.server.Downloads, jc.DeepEquals, []*charm.URL{charmURL})
	s.assertCached(c, charmURL)
}

func (s *legacyCharmStoreSuite) TestGetBadCache(c *gc.C) {
	c.Assert(os.Mkdir(filepath.Join(charmrepo.CacheDir, "cache"), 0777), gc.IsNil)
	base := "cs:series/good"
	charmURL := charm.MustParseURL(base)
	revCharmURL := charm.MustParseURL(base + "-23")
	name := charm.Quote(revCharmURL.String()) + ".charm"
	err := ioutil.WriteFile(filepath.Join(charmrepo.CacheDir, "cache", name), nil, 0666)
	c.Assert(err, gc.IsNil)
	ch, err := s.store.Get(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch, gc.NotNil)
	c.Assert(s.server.Downloads, jc.DeepEquals, []*charm.URL{revCharmURL})
	s.assertCached(c, charmURL)
	s.assertCached(c, revCharmURL)
}

func (s *legacyCharmStoreSuite) TestGetTestModeFlag(c *gc.C) {
	base := "cs:series/good-12"
	charmURL := charm.MustParseURL(base)
	ch, err := s.store.Get(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch, gc.NotNil)
	c.Assert(s.server.Downloads, jc.DeepEquals, []*charm.URL{charmURL})
	c.Assert(s.server.DownloadsNoStats, gc.IsNil)
	c.Assert(s.server.InfoRequestCount, gc.Equals, 1)
	c.Assert(s.server.InfoRequestCountNoStats, gc.Equals, 0)

	storeInTestMode := s.store.WithTestMode(true)
	other := "cs:series/good-23"
	otherURL := charm.MustParseURL(other)
	ch, err = storeInTestMode.Get(otherURL)
	c.Assert(err, gc.IsNil)
	c.Assert(ch, gc.NotNil)
	c.Assert(s.server.Downloads, jc.DeepEquals, []*charm.URL{charmURL})
	c.Assert(s.server.DownloadsNoStats, jc.DeepEquals, []*charm.URL{otherURL})
	c.Assert(s.server.InfoRequestCount, gc.Equals, 1)
	c.Assert(s.server.InfoRequestCountNoStats, gc.Equals, 1)
}

// The following tests cover the low-level CharmStore-specific API.

func (s *legacyCharmStoreSuite) TestInfo(c *gc.C) {
	charmURLs := []charm.Location{
		charm.MustParseURL("cs:series/good"),
		charm.MustParseURL("cs:series/better"),
		charm.MustParseURL("cs:series/best"),
	}
	infos, err := s.store.Info(charmURLs...)
	c.Assert(err, gc.IsNil)
	c.Assert(infos, gc.HasLen, 3)
	expected := []int{23, 24, 25}
	for i, info := range infos {
		c.Assert(info.Errors, gc.IsNil)
		c.Assert(info.Revision, gc.Equals, expected[i])
	}
}

func (s *legacyCharmStoreSuite) TestInfoNotFound(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/missing")
	info, err := s.store.Info(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(info, gc.HasLen, 1)
	c.Assert(info[0].Errors, gc.HasLen, 1)
	c.Assert(info[0].Errors[0], gc.Matches, `charm not found: cs:series/missing`)
}

func (s *legacyCharmStoreSuite) TestInfoError(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/borken")
	info, err := s.store.Info(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(info, gc.HasLen, 1)
	c.Assert(info[0].Errors, jc.DeepEquals, []string{"badness"})
}

func (s *legacyCharmStoreSuite) TestInfoWarning(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/unwise")
	info, err := s.store.Info(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(info, gc.HasLen, 1)
	c.Assert(info[0].Warnings, jc.DeepEquals, []string{"foolishness"})
}

func (s *legacyCharmStoreSuite) TestInfoTestModeFlag(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/good")
	_, err := s.store.Info(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(s.server.InfoRequestCount, gc.Equals, 1)
	c.Assert(s.server.InfoRequestCountNoStats, gc.Equals, 0)

	storeInTestMode, ok := s.store.WithTestMode(true).(*charmrepo.LegacyCharmStore)
	c.Assert(ok, gc.Equals, true)
	_, err = storeInTestMode.Info(charmURL)
	c.Assert(err, gc.IsNil)
	c.Assert(s.server.InfoRequestCount, gc.Equals, 1)
	c.Assert(s.server.InfoRequestCountNoStats, gc.Equals, 1)
}

func (s *legacyCharmStoreSuite) TestInfoDNSError(c *gc.C) {
	store := newLegacyStore("http://127.1.2.3")
	charmURL := charm.MustParseURL("cs:series/good")
	resp, err := store.Info(charmURL)
	c.Assert(resp, gc.IsNil)
	expect := `Cannot access the charm store. .*`
	c.Assert(err, gc.ErrorMatches, expect)
}

func (s *legacyCharmStoreSuite) TestEvent(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/good")
	event, err := s.store.Event(charmURL, "")
	c.Assert(err, gc.IsNil)
	c.Assert(event.Errors, gc.IsNil)
	c.Assert(event.Revision, gc.Equals, 23)
	c.Assert(event.Digest, gc.Equals, "the-digest")
}

func (s *legacyCharmStoreSuite) TestEventWithDigest(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/good")
	event, err := s.store.Event(charmURL, "the-digest")
	c.Assert(err, gc.IsNil)
	c.Assert(event.Errors, gc.IsNil)
	c.Assert(event.Revision, gc.Equals, 23)
	c.Assert(event.Digest, gc.Equals, "the-digest")
}

func (s *legacyCharmStoreSuite) TestEventNotFound(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/missing")
	event, err := s.store.Event(charmURL, "")
	c.Assert(err, gc.ErrorMatches, `charm event not found for "cs:series/missing"`)
	c.Assert(event, gc.IsNil)
}

func (s *legacyCharmStoreSuite) TestEventNotFoundDigest(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/good")
	event, err := s.store.Event(charmURL, "missing-digest")
	c.Assert(err, gc.ErrorMatches, `charm event not found for "cs:series/good" with digest "missing-digest"`)
	c.Assert(event, gc.IsNil)
}

func (s *legacyCharmStoreSuite) TestEventError(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/borken")
	event, err := s.store.Event(charmURL, "")
	c.Assert(err, gc.IsNil)
	c.Assert(event.Errors, jc.DeepEquals, []string{"badness"})
}

func (s *legacyCharmStoreSuite) TestAuthorization(c *gc.C) {
	store := s.store.WithAuthAttrs("token=value")

	base := "cs:series/good"
	charmURL := charm.MustParseURL(base)
	_, err := store.Get(charmURL)

	c.Assert(err, gc.IsNil)

	c.Assert(s.server.Authorizations, gc.HasLen, 1)
	c.Assert(s.server.Authorizations[0], gc.Equals, "charmstore token=value")
}

func (s *legacyCharmStoreSuite) TestNilAuthorization(c *gc.C) {
	store := s.store.WithAuthAttrs("")

	base := "cs:series/good"
	charmURL := charm.MustParseURL(base)
	_, err := store.Get(charmURL)

	c.Assert(err, gc.IsNil)
	c.Assert(s.server.Authorizations, gc.HasLen, 0)
}

func (s *legacyCharmStoreSuite) TestMetadata(c *gc.C) {
	store := s.store.WithJujuAttrs("juju-metadata")

	base := "cs:series/good"
	charmURL := charm.MustParseURL(base)
	_, err := store.Get(charmURL)

	c.Assert(err, gc.IsNil)
	c.Assert(s.server.Metadata, gc.HasLen, 1)
	c.Assert(s.server.Metadata[0], gc.Equals, "juju-metadata")
}

func (s *legacyCharmStoreSuite) TestNilMetadata(c *gc.C) {
	base := "cs:series/good"
	charmURL := charm.MustParseURL(base)
	_, err := s.store.Get(charmURL)

	c.Assert(err, gc.IsNil)
	c.Assert(s.server.Metadata, gc.HasLen, 0)
}

func (s *legacyCharmStoreSuite) TestEventWarning(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/unwise")
	event, err := s.store.Event(charmURL, "")
	c.Assert(err, gc.IsNil)
	c.Assert(event.Warnings, jc.DeepEquals, []string{"foolishness"})
}

func (s *legacyCharmStoreSuite) TestBranchLocation(c *gc.C) {
	charmURL := charm.MustParseURL("cs:series/name")
	location := s.store.BranchLocation(charmURL)
	c.Assert(location, gc.Equals, "lp:charms/series/name")

	charmURL = charm.MustParseURL("cs:~user/series/name")
	location = s.store.BranchLocation(charmURL)
	c.Assert(location, gc.Equals, "lp:~user/charms/series/name/trunk")
}

func (s *legacyCharmStoreSuite) TestCharmURL(c *gc.C) {
	tests := []struct{ url, loc string }{
		{"cs:precise/wordpress", "lp:charms/precise/wordpress"},
		{"cs:precise/wordpress", "http://launchpad.net/+branch/charms/precise/wordpress"},
		{"cs:precise/wordpress", "https://launchpad.net/+branch/charms/precise/wordpress"},
		{"cs:precise/wordpress", "http://code.launchpad.net/+branch/charms/precise/wordpress"},
		{"cs:precise/wordpress", "https://code.launchpad.net/+branch/charms/precise/wordpress"},
		{"cs:precise/wordpress", "bzr+ssh://bazaar.launchpad.net/+branch/charms/precise/wordpress"},
		{"cs:~charmers/precise/wordpress", "lp:~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "http://launchpad.net/~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "https://launchpad.net/~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "http://code.launchpad.net/~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "https://code.launchpad.net/~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "http://launchpad.net/+branch/~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "https://launchpad.net/+branch/~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "http://code.launchpad.net/+branch/~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "https://code.launchpad.net/+branch/~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "bzr+ssh://bazaar.launchpad.net/~charmers/charms/precise/wordpress/trunk"},
		{"cs:~charmers/precise/wordpress", "bzr+ssh://bazaar.launchpad.net/~charmers/charms/precise/wordpress/trunk/"},
		{"cs:~charmers/precise/wordpress", "~charmers/charms/precise/wordpress/trunk"},
		{"", "lp:~charmers/charms/precise/wordpress/whatever"},
		{"", "lp:~charmers/whatever/precise/wordpress/trunk"},
		{"", "lp:whatever/precise/wordpress"},
	}
	for _, t := range tests {
		charmURL, err := s.store.CharmURL(t.loc)
		if t.url == "" {
			c.Assert(err, gc.ErrorMatches, fmt.Sprintf("unknown branch location: %q", t.loc))
		} else {
			c.Assert(err, gc.IsNil)
			c.Assert(charmURL.String(), gc.Equals, t.url)
		}
	}
}

func newLegacyStore(url string) *charmrepo.LegacyCharmStore {
	return &charmrepo.LegacyCharmStore{BaseURL: url}
}
