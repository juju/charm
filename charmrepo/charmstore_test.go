// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmrepo_test

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http/httptest"
	"os"

	jujutesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charmstore.v4"
	"gopkg.in/juju/charmstore.v4/csclient"
	"gopkg.in/mgo.v2"

	"gopkg.in/juju/charm.v5-unstable"
	"gopkg.in/juju/charm.v5-unstable/charmrepo"
	charmtesting "gopkg.in/juju/charm.v5-unstable/testing"
)

type charmStoreSuite struct {
	jujutesting.IsolationSuite
}

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

var serverParams = charmstore.ServerParams{
	AuthUsername: "test-user",
	AuthPassword: "test-password",
}

type charmStoreBaseSuite struct {
	charmtesting.IsolatedMgoSuite
	srv  *httptest.Server
	repo charmrepo.Interface
}

var _ = gc.Suite(&charmStoreBaseSuite{})

func (s *charmStoreBaseSuite) SetUpTest(c *gc.C) {
	s.IsolatedMgoSuite.SetUpTest(c)
	s.srv = newServer(c, s.Session)
	s.repo = newStoreRepo(c, s.srv.URL)
}

func (s *charmStoreBaseSuite) TearDownTest(c *gc.C) {
	s.srv.Close()
	s.IsolatedMgoSuite.TearDownTest(c)
}

// addCharm uploads a charm to the testing charm store, and returns the
// resulting charm URL.
func (s *charmStoreBaseSuite) addCharm(c *gc.C, url, name string) *charm.URL {
	client := csclient.New(csclient.Params{
		URL:      s.srv.URL,
		User:     serverParams.AuthUsername,
		Password: serverParams.AuthPassword,
	})
	id, err := client.UploadCharm(
		charm.MustParseReference(url),
		TestCharms.CharmDir(name))
	c.Assert(err, jc.ErrorIsNil)
	return (*charm.URL)(id)
}

type charmStoreRepoSuite struct {
	charmStoreBaseSuite
}

var _ = gc.Suite(&charmStoreRepoSuite{})

func (s *charmStoreRepoSuite) TestLatest(c *gc.C) {
	// Add some charms to the charm store.
	s.addCharm(c, "~who/trusty/mysql", "mysql")
	s.addCharm(c, "~who/precise/wordpress", "wordpress")
	// Use different charms so that revision is actually increased
	s.addCharm(c, "~dalek/trusty/riak", "wordpress")
	s.addCharm(c, "~dalek/trusty/riak", "riak")
	s.addCharm(c, "~dalek/trusty/riak", "wordpress")
	s.addCharm(c, "~dalek/trusty/riak", "riak")

	// Calculate and store the expected hashes for re uploaded charms.
	mysqlHash := hashOfCharm(c, "mysql")
	wordpressHash := hashOfCharm(c, "wordpress")
	riakHash := hashOfCharm(c, "riak")

	// Define the tests to be run.
	tests := []struct {
		about string
		urls  []*charm.URL
		revs  []charmrepo.CharmRevision
	}{{
		about: "no urls",
	}, {
		about: "charm not found",
		urls:  []*charm.URL{charm.MustParseURL("cs:trusty/no-such-42")},
		revs: []charmrepo.CharmRevision{{
			Err: charmrepo.CharmNotFound("cs:trusty/no-such"),
		}},
	}, {
		about: "resolve",
		urls: []*charm.URL{
			charm.MustParseURL("cs:~who/trusty/mysql-42"),
			charm.MustParseURL("cs:~who/trusty/mysql-0"),
			charm.MustParseURL("cs:~who/trusty/mysql"),
		},
		revs: []charmrepo.CharmRevision{{
			Revision: 0,
			Sha256:   mysqlHash,
		}, {
			Revision: 0,
			Sha256:   mysqlHash,
		}, {
			Revision: 0,
			Sha256:   mysqlHash,
		}},
	}, {
		about: "multiple charms",
		urls: []*charm.URL{
			charm.MustParseURL("cs:~who/precise/wordpress"),
			charm.MustParseURL("cs:~who/trusty/mysql-47"),
			charm.MustParseURL("cs:~dalek/trusty/no-such"),
			charm.MustParseURL("cs:~dalek/trusty/riak-0"),
		},
		revs: []charmrepo.CharmRevision{{
			Revision: 0,
			Sha256:   wordpressHash,
		}, {
			Revision: 0,
			Sha256:   mysqlHash,
		}, {
			Err: charmrepo.CharmNotFound("cs:~dalek/trusty/no-such"),
		}, {
			Revision: 3,
			Sha256:   riakHash,
		}},
	}}

	// Run the tests.
	for i, test := range tests {
		c.Logf("test %d: %s", i, test.about)
		revs, err := s.repo.Latest(test.urls...)
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(revs, jc.DeepEquals, test.revs)
	}
}

func (s *charmStoreRepoSuite) TestResolve(c *gc.C) {
	// Add some charms to the charm store.
	s.addCharm(c, "~who/trusty/mysql", "mysql")
	// Use different charms so that revision is actually increased
	s.addCharm(c, "~who/precise/wordpress", "logging")
	s.addCharm(c, "~who/precise/wordpress", "wordpress")
	s.addCharm(c, "~dalek/utopic/riak", "wordpress")
	s.addCharm(c, "~dalek/utopic/riak", "riak")
	s.addCharm(c, "~dalek/utopic/riak", "wordpress")
	s.addCharm(c, "~dalek/utopic/riak", "riak")

	// Define the tests to be run.
	tests := []struct {
		id  string
		url string
		err string
	}{{
		id:  "~who/mysql",
		url: "cs:~who/trusty/mysql",
	}, {
		id:  "~who/trusty/mysql",
		url: "cs:~who/trusty/mysql",
	}, {
		id:  "~who/wordpress",
		url: "cs:~who/precise/wordpress",
	}, {
		id:  "~who/wordpress-1",
		url: "cs:~who/precise/wordpress-1",
	}, {
		id:  "~dalek/riak",
		url: "cs:~dalek/utopic/riak",
	}, {
		id:  "~dalek/utopic/riak-2",
		url: "cs:~dalek/utopic/riak-2",
	}, {
		id:  "no-such",
		err: `cannot get metadata from the charm store: cannot get "/no-such/meta/any?include=id-series": no matching charm or bundle for "cs:no-such"`,
	}}

	// Run the tests.
	for i, test := range tests {
		c.Logf("test %d: %s", i, test.id)
		url, err := s.repo.Resolve(charm.MustParseReference(test.id))
		if test.err != "" {
			c.Assert(err.Error(), gc.Equals, test.err)
			c.Assert(url, gc.IsNil)
			continue
		}
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(url, jc.DeepEquals, charm.MustParseURL(test.url))
	}
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

// newServer instantiates a new charm store server.
func newServer(c *gc.C, session *mgo.Session) *httptest.Server {
	db := session.DB("charm-testing")
	handler, err := charmstore.NewServer(db, nil, "", serverParams, charmstore.V4)
	c.Assert(err, jc.ErrorIsNil)
	return httptest.NewServer(handler)
}

// hashOfCharm returns the SHA256 hash sum for the given charm name.
func hashOfCharm(c *gc.C, name string) string {
	path := TestCharms.CharmArchivePath(c.MkDir(), name)
	f, err := os.Open(path)
	c.Assert(err, jc.ErrorIsNil)
	defer f.Close()
	hash := sha256.New()
	_, err = io.Copy(hash, f)
	c.Assert(err, jc.ErrorIsNil)
	return fmt.Sprintf("%x", hash.Sum(nil))
}
