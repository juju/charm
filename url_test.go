// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm_test

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/juju/charm.v3"
	"gopkg.in/mgo.v2/bson"
	gc "launchpad.net/gocheck"
)

type URLSuite struct{}

var _ = gc.Suite(&URLSuite{})

var urlTests = []struct {
	s, err string
	exact  string
	ref    *charm.Reference
}{{
	s:   "cs:~user/series/name",
	ref: &charm.Reference{"cs", "user", "name", -1, "series"},
}, {
	s:   "cs:~user/series/name-0",
	ref: &charm.Reference{"cs", "user", "name", 0, "series"},
}, {
	s:   "cs:series/name",
	ref: &charm.Reference{"cs", "", "name", -1, "series"},
}, {
	s:   "cs:series/name-42",
	ref: &charm.Reference{"cs", "", "name", 42, "series"},
}, {
	s:   "local:series/name-1",
	ref: &charm.Reference{"local", "", "name", 1, "series"},
}, {
	s:   "local:series/name",
	ref: &charm.Reference{"local", "", "name", -1, "series"},
}, {
	s:   "local:series/n0-0n-n0",
	ref: &charm.Reference{"local", "", "n0-0n-n0", -1, "series"},
}, {
	s:   "cs:~user/name",
	ref: &charm.Reference{"cs", "user", "name", -1, ""},
}, {
	s:   "cs:name",
	ref: &charm.Reference{"cs", "", "name", -1, ""},
}, {
	s:   "local:name",
	ref: &charm.Reference{"local", "", "name", -1, ""},
}, {
	s:   "bs:~user/series/name-1",
	err: "charm URL has invalid schema: .*",
}, {
	s: ":foo",
	err: "charm URL has invalid schema: .*",
}, {
	s:   "cs:~1/series/name-1",
	err: "charm URL has invalid user name: .*",
}, {
	s:   "cs:~user",
	err: "charm URL without charm name: .*",
}, {
	s:   "cs:~user/1/name-1",
	err: "charm URL has invalid series: .*",
}, {
	s:   "cs:~user/series/name-1-2",
	err: "charm URL has invalid charm name: .*",
}, {
	s:   "cs:~user/series/name-1-name-2",
	err: "charm URL has invalid charm name: .*",
}, {
	s:   "cs:~user/series/name--name-2",
	err: "charm URL has invalid charm name: .*",
}, {
	s:   "cs:foo-1-2",
	err: "charm URL has invalid charm name: .*",
}, {
	s:   "cs:~user/series/huh/name-1",
	err: "charm URL has invalid form: .*",
}, {
	s:   "cs:/name",
	err: "charm URL has invalid series: .*",
}, {
	s:   "local:~user/series/name",
	err: "local charm URL with user name: .*",
}, {
	s:   "local:~user/name",
	err: "local charm URL with user name: .*",
}, {
	s:     "precise/wordpress",
	exact: "cs:precise/wordpress",
	ref:   &charm.Reference{"cs", "", "wordpress", -1, "precise"},
	err:   `charm URL has no schema: "precise/wordpress"`,
}, {
	s:     "foo",
	exact: "cs:foo",
	ref:   &charm.Reference{"cs", "", "foo", -1, ""},
}, {
	s:     "foo-1",
	exact: "cs:foo-1",
	ref:   &charm.Reference{"cs", "", "foo", 1, ""},
}, {
	s:     "n0-n0-n0",
	exact: "cs:n0-n0-n0",
	ref:   &charm.Reference{"cs", "", "n0-n0-n0", -1, ""},
}, {
	s:     "cs:foo",
	exact: "cs:foo",
	ref:   &charm.Reference{"cs", "", "foo", -1, ""},
}, {
	s:     "local:foo",
	exact: "local:foo",
	ref:   &charm.Reference{"local", "", "foo", -1, ""},
}, {
	s:     "series/foo",
	exact: "cs:series/foo",
	ref:   &charm.Reference{"cs", "", "foo", -1, "series"},
	err:   `charm URL has no schema: "series/foo"`,
}, {
	s:   "series/foo/bar",
	err: `charm URL has invalid form: "series/foo/bar"`,
}, {
	s:   "cs:foo/~blah",
	err: `charm URL has invalid charm name: "cs:foo/~blah"`,
}}

func (s *URLSuite) TestParseURL(c *gc.C) {
	for i, t := range urlTests {
		c.Logf("test %d: %q", i, t.s)
		url, uerr := charm.ParseURL(t.s)
		ref, rerr := charm.ParseReference(t.s)

		expectStr := t.s
		if t.exact != "" {
			expectStr = t.exact
		}
		if t.ref != nil {
			// ParseReference, at least, should have succeeded.
			c.Assert(rerr, gc.IsNil)
			c.Assert(ref, gc.DeepEquals, t.ref)
			c.Check(ref.String(), gc.Equals, expectStr)
		}
		if t.err != "" {
			c.Assert(uerr, gc.ErrorMatches, t.err)
			c.Assert(url, gc.IsNil)
			if t.ref == nil {
				c.Assert(rerr, gc.NotNil)
				// Errors from both ParseURL and ParseReference should match.
				c.Check(uerr.Error(), gc.Equals, rerr.Error())
				c.Check(ref, gc.IsNil)
			}
			continue
		}
		if t.ref.Series == "" {
			// ParseURL with an empty series should report an unresolved error.
			c.Assert(url, gc.IsNil)
			c.Assert(uerr, gc.Equals, charm.ErrUnresolvedUrl)
			continue
		}
		// When ParseURL succeeds, it should return the same thing
		// as ParseReference.
		c.Assert(uerr, gc.IsNil)
		c.Check(url.Reference(), gc.DeepEquals, ref)

		// URL parsing should always be reversible.
		c.Check(url.String(), gc.Equals, t.s)
	}
}

var inferTests = []struct {
	vague, exact string
}{
	{"foo", "cs:defseries/foo"},
	{"foo-1", "cs:defseries/foo-1"},
	{"n0-n0-n0", "cs:defseries/n0-n0-n0"},
	{"cs:foo", "cs:defseries/foo"},
	{"local:foo", "local:defseries/foo"},
	{"series/foo", "cs:series/foo"},
	{"cs:series/foo", "cs:series/foo"},
	{"local:series/foo", "local:series/foo"},
	{"cs:~user/foo", "cs:~user/defseries/foo"},
	{"cs:~user/series/foo", "cs:~user/series/foo"},
	{"local:~user/series/foo", "local:~user/series/foo"},
	{"bs:foo", "bs:defseries/foo"},
	{"cs:~1/foo", "cs:~1/defseries/foo"},
	{"cs:foo-1-2", "cs:defseries/foo-1-2"},
}

func (s *URLSuite) TestInferURL(c *gc.C) {
	for i, t := range inferTests {
		c.Logf("test %d", i)
		comment := gc.Commentf("InferURL(%q, %q)", t.vague, "defseries")
		inferred, ierr := charm.InferURL(t.vague, "defseries")
		parsed, perr := charm.ParseURL(t.exact)
		if perr == nil {
			c.Check(inferred, gc.DeepEquals, parsed, comment)
			c.Check(ierr, gc.IsNil)
		} else {
			expect := perr.Error()
			if t.vague != t.exact {
				if colIdx := strings.Index(expect, ":"); colIdx > 0 {
					expect = expect[:colIdx]
				}
			}
			c.Check(ierr.Error(), gc.Matches, expect+".*", comment)
		}
	}
	u, err := charm.InferURL("~blah", "defseries")
	c.Assert(u, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, "charm URL without charm name: .*")
}

var inferNoDefaultSeriesTests = []struct {
	vague, exact string
	resolved     bool
}{
	{"foo", "", false},
	{"foo-1", "", false},
	{"cs:foo", "", false},
	{"cs:~user/foo", "", false},
	{"series/foo", "cs:series/foo", true},
	{"cs:series/foo", "cs:series/foo", true},
	{"cs:~user/series/foo", "cs:~user/series/foo", true},
}

func (s *URLSuite) TestInferURLNoDefaultSeries(c *gc.C) {
	for i, t := range inferNoDefaultSeriesTests {
		c.Logf("%d: %s", i, t.vague)
		inferred, err := charm.InferURL(t.vague, "")
		if t.exact == "" {
			c.Assert(err, gc.ErrorMatches, fmt.Sprintf("cannot infer charm URL for %q: charm url series is not resolved", t.vague))
		} else {
			parsed, err := charm.ParseURL(t.exact)
			c.Assert(err, gc.IsNil)
			c.Assert(inferred, gc.DeepEquals, parsed, gc.Commentf(`InferURL(%q, "")`, t.vague))
		}
	}
}

var validTests = []struct {
	valid  func(string) bool
	string string
	expect bool
}{

	{charm.IsValidName, "", false},
	{charm.IsValidName, "wordpress", true},
	{charm.IsValidName, "Wordpress", false},
	{charm.IsValidName, "word-press", true},
	{charm.IsValidName, "word press", false},
	{charm.IsValidName, "word^press", false},
	{charm.IsValidName, "-wordpress", false},
	{charm.IsValidName, "wordpress-", false},
	{charm.IsValidName, "wordpress2", true},
	{charm.IsValidName, "wordpress-2", false},
	{charm.IsValidName, "word2-press2", true},

	{charm.IsValidSeries, "", false},
	{charm.IsValidSeries, "precise", true},
	{charm.IsValidSeries, "Precise", false},
	{charm.IsValidSeries, "pre cise", false},
	{charm.IsValidSeries, "pre-cise", false},
	{charm.IsValidSeries, "pre^cise", false},
	{charm.IsValidSeries, "prec1se", true},
	{charm.IsValidSeries, "-precise", false},
	{charm.IsValidSeries, "precise-", false},
	{charm.IsValidSeries, "precise-1", false},
	{charm.IsValidSeries, "precise1", true},
	{charm.IsValidSeries, "pre-c1se", false},
}

func (s *URLSuite) TestValidCheckers(c *gc.C) {
	for i, t := range validTests {
		c.Logf("test %d: %s", i, t.string)
		c.Assert(t.valid(t.string), gc.Equals, t.expect, gc.Commentf("%s", t.string))
	}
}

func (s *URLSuite) TestMustParseURL(c *gc.C) {
	url := charm.MustParseURL("cs:series/name")
	c.Assert(url, gc.DeepEquals, &charm.URL{"cs", "", "name", -1, "series"})
	f := func() { charm.MustParseURL("local:@@/name") }
	c.Assert(f, gc.PanicMatches, "charm URL has invalid series: .*")
	f = func() { charm.MustParseURL("cs:~user") }
	c.Assert(f, gc.PanicMatches, "charm URL without charm name: .*")
	f = func() { charm.MustParseURL("cs:~user") }
	c.Assert(f, gc.PanicMatches, "charm URL without charm name: .*")
	f = func() { charm.MustParseURL("cs:name") }
	c.Assert(f, gc.PanicMatches, "charm url series is not resolved")
}

func (s *URLSuite) TestWithRevision(c *gc.C) {
	url := charm.MustParseURL("cs:series/name")
	other := url.WithRevision(1)
	c.Assert(url, gc.DeepEquals, &charm.URL{"cs", "", "name", -1, "series"})
	c.Assert(other, gc.DeepEquals, &charm.URL{"cs", "", "name", 1, "series"})

	// Should always copy. The opposite behavior is error prone.
	c.Assert(other.WithRevision(1), gc.Not(gc.Equals), other)
	c.Assert(other.WithRevision(1), gc.DeepEquals, other)
}

var codecs = []struct {
	Marshal   func(interface{}) ([]byte, error)
	Unmarshal func([]byte, interface{}) error
}{{
	Marshal:   bson.Marshal,
	Unmarshal: bson.Unmarshal,
}, {
	Marshal:   json.Marshal,
	Unmarshal: json.Unmarshal,
}}

func (s *URLSuite) TestURLCodecs(c *gc.C) {
	for i, codec := range codecs {
		c.Logf("codec %d", i)
		type doc struct {
			URL *charm.URL
			Ref *charm.Reference
		}
		url := charm.MustParseURL("cs:series/name")
		v0 := doc{url, url.Reference()}
		data, err := codec.Marshal(v0)
		c.Assert(err, gc.IsNil)
		var v doc
		err = codec.Unmarshal(data, &v)
		c.Assert(v, gc.DeepEquals, v0)

		// Check that the underlying representation
		// is a string.
		type strDoc struct {
			URL string
			Ref string
		}
		var vs strDoc
		err = codec.Unmarshal(data, &vs)
		c.Assert(err, gc.IsNil)
		c.Assert(vs.URL, gc.Equals, "cs:series/name")
		c.Assert(vs.Ref, gc.Equals, "cs:series/name")

		data, err = codec.Marshal(doc{})
		c.Assert(err, gc.IsNil)
		err = codec.Unmarshal(data, &v)
		c.Assert(err, gc.IsNil)
		c.Assert(v.URL, gc.IsNil)
		c.Assert(v.Ref, gc.IsNil)
	}
}

func (s *URLSuite) TestJSONGarbage(c *gc.C) {
	// unmarshalling json gibberish
	for _, value := range []string{":{", `"cs:{}+<"`, `"cs:~_~/f00^^&^/baaaar$%-?"`} {
		err := json.Unmarshal([]byte(value), new(struct{ URL *charm.URL }))
		c.Check(err, gc.NotNil)
		err = json.Unmarshal([]byte(value), new(struct{ Ref *charm.Reference }))
		c.Check(err, gc.NotNil)
	}
}

type QuoteSuite struct{}

var _ = gc.Suite(&QuoteSuite{})

func (s *QuoteSuite) TestUnmodified(c *gc.C) {
	// Check that a string containing only valid
	// chars stays unmodified.
	in := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-"
	out := charm.Quote(in)
	c.Assert(out, gc.Equals, in)
}

func (s *QuoteSuite) TestQuote(c *gc.C) {
	// Check that invalid chars are translated correctly.
	in := "hello_there/how'are~you-today.sir"
	out := charm.Quote(in)
	c.Assert(out, gc.Equals, "hello_5f_there_2f_how_27_are_7e_you-today.sir")
}
