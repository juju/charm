// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	gc "gopkg.in/check.v1"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/yaml.v2"

	"github.com/juju/charm/v7"
)

type URLSuite struct{}

var _ = gc.Suite(&URLSuite{})

var urlTests = []struct {
	s, err string
	exact  string
	url    *charm.URL
}{{
	s:   "cs:~user/series/name",
	url: &charm.URL{"cs", "user", "name", -1, "series"},
}, {
	s:   "cs:~user/series/name-0",
	url: &charm.URL{"cs", "user", "name", 0, "series"},
}, {
	s:   "cs:series/name",
	url: &charm.URL{"cs", "", "name", -1, "series"},
}, {
	s:   "cs:series/name-42",
	url: &charm.URL{"cs", "", "name", 42, "series"},
}, {
	s:   "local:series/name-1",
	url: &charm.URL{"local", "", "name", 1, "series"},
}, {
	s:   "local:series/name",
	url: &charm.URL{"local", "", "name", -1, "series"},
}, {
	s:   "local:series/n0-0n-n0",
	url: &charm.URL{"local", "", "n0-0n-n0", -1, "series"},
}, {
	s:   "cs:~user/name",
	url: &charm.URL{"cs", "user", "name", -1, ""},
}, {
	s:   "cs:name",
	url: &charm.URL{"cs", "", "name", -1, ""},
}, {
	s:   "local:name",
	url: &charm.URL{"local", "", "name", -1, ""},
}, {
	s:     "http://jujucharms.com/u/user/name/series/1",
	url:   &charm.URL{"cs", "user", "name", 1, "series"},
	exact: "cs:~user/series/name-1",
}, {
	s:     "http://www.jujucharms.com/u/user/name/series/1",
	url:   &charm.URL{"cs", "user", "name", 1, "series"},
	exact: "cs:~user/series/name-1",
}, {
	s:     "https://www.jujucharms.com/u/user/name/series/1",
	url:   &charm.URL{"cs", "user", "name", 1, "series"},
	exact: "cs:~user/series/name-1",
}, {
	s:     "https://jujucharms.com/u/user/name/series/1",
	url:   &charm.URL{"cs", "user", "name", 1, "series"},
	exact: "cs:~user/series/name-1",
}, {
	s:     "https://jujucharms.com/u/user/name/series",
	url:   &charm.URL{"cs", "user", "name", -1, "series"},
	exact: "cs:~user/series/name",
}, {
	s:     "https://jujucharms.com/u/user/name/1",
	url:   &charm.URL{"cs", "user", "name", 1, ""},
	exact: "cs:~user/name-1",
}, {
	s:     "https://jujucharms.com/u/user/name",
	url:   &charm.URL{"cs", "user", "name", -1, ""},
	exact: "cs:~user/name",
}, {
	s:     "https://jujucharms.com/name",
	url:   &charm.URL{"cs", "", "name", -1, ""},
	exact: "cs:name",
}, {
	s:     "https://jujucharms.com/name/series",
	url:   &charm.URL{"cs", "", "name", -1, "series"},
	exact: "cs:series/name",
}, {
	s:     "https://jujucharms.com/name/1",
	url:   &charm.URL{"cs", "", "name", 1, ""},
	exact: "cs:name-1",
}, {
	s:     "https://jujucharms.com/name/series/1",
	url:   &charm.URL{"cs", "", "name", 1, "series"},
	exact: "cs:series/name-1",
}, {
	s:     "https://jujucharms.com/u/user/name/series/1/",
	url:   &charm.URL{"cs", "user", "name", 1, "series"},
	exact: "cs:~user/series/name-1",
}, {
	s:     "https://jujucharms.com/u/user/name/series/",
	url:   &charm.URL{"cs", "user", "name", -1, "series"},
	exact: "cs:~user/series/name",
}, {
	s:     "https://jujucharms.com/u/user/name/1/",
	url:   &charm.URL{"cs", "user", "name", 1, ""},
	exact: "cs:~user/name-1",
}, {
	s:     "https://jujucharms.com/u/user/name/",
	url:   &charm.URL{"cs", "user", "name", -1, ""},
	exact: "cs:~user/name",
}, {
	s:     "https://jujucharms.com/name/",
	url:   &charm.URL{"cs", "", "name", -1, ""},
	exact: "cs:name",
}, {
	s:     "https://jujucharms.com/name/series/",
	url:   &charm.URL{"cs", "", "name", -1, "series"},
	exact: "cs:series/name",
}, {
	s:     "https://jujucharms.com/name/1/",
	url:   &charm.URL{"cs", "", "name", 1, ""},
	exact: "cs:name-1",
}, {
	s:     "https://jujucharms.com/name/series/1/",
	url:   &charm.URL{"cs", "", "name", 1, "series"},
	exact: "cs:series/name-1",
}, {
	s:   "https://jujucharms.com/",
	err: `cannot parse URL $URL: name "" not valid`,
}, {
	s:   "https://jujucharms.com/bad.wolf",
	err: `cannot parse URL $URL: name "bad.wolf" not valid`,
}, {
	s:   "https://jujucharms.com/u/",
	err: "charm or bundle URL $URL malformed, expected \"/u/<user>/<name>\"",
}, {
	s:   "https://jujucharms.com/u/badwolf",
	err: "charm or bundle URL $URL malformed, expected \"/u/<user>/<name>\"",
}, {
	s:   "https://jujucharms.com/name/series/badwolf",
	err: "charm or bundle URL has malformed revision: \"badwolf\" in $URL",
}, {
	s:   "https://jujucharms.com/name/bad.wolf/42",
	err: `cannot parse URL $URL: series name "bad.wolf" not valid`,
}, {
	s:   "https://badwolf@jujucharms.com/name/series/42",
	err: `charm or bundle URL $URL has unrecognized parts`,
}, {
	s:   "https://jujucharms.com/name/series/42#bad-wolf",
	err: `charm or bundle URL $URL has unrecognized parts`,
}, {
	s:   "https://jujucharms.com/name/series/42?bad=wolf",
	err: `charm or bundle URL $URL has unrecognized parts`,
}, {
	s:   "bs:~user/series/name-1",
	err: `cannot parse URL $URL: schema "bs" not valid`,
}, {
	s:   ":foo",
	err: `cannot parse charm or bundle URL: $URL`,
}, {
	s:   "cs:~1/series/name-1",
	err: `charm or bundle URL has invalid user name: $URL`,
}, {
	s:   "cs:~user",
	err: `URL without charm or bundle name: $URL`,
}, {
	s:   "cs:~user/1/name-1",
	err: `cannot parse URL $URL: series name "1" not valid`,
}, {
	s:   "cs:~user/series/name-1-2",
	err: `cannot parse URL $URL: name "name-1" not valid`,
}, {
	s:   "cs:~user/series/name-1-name-2",
	err: `cannot parse URL $URL: name "name-1-name" not valid`,
}, {
	s:   "cs:~user/series/name--name-2",
	err: `cannot parse URL $URL: name "name--name" not valid`,
}, {
	s:   "cs:foo-1-2",
	err: `cannot parse URL $URL: name "foo-1" not valid`,
}, {
	s:   "cs:~user/series/huh/name-1",
	err: `charm or bundle URL has invalid form: $URL`,
}, {
	s:   "cs:~user/production/series/name-1",
	err: `charm or bundle URL has invalid form: $URL`,
}, {
	s:   "cs:/name",
	err: `cannot parse URL $URL: series name "" not valid`,
}, {
	s:   "local:~user/series/name",
	err: `local charm or bundle URL with user name: $URL`,
}, {
	s:   "local:~user/name",
	err: `local charm or bundle URL with user name: $URL`,
}, {
	s:     "precise/wordpress",
	exact: "cs:precise/wordpress",
	url:   &charm.URL{"cs", "", "wordpress", -1, "precise"},
}, {
	s:     "foo",
	exact: "cs:foo",
	url:   &charm.URL{"cs", "", "foo", -1, ""},
}, {
	s:     "foo-1",
	exact: "cs:foo-1",
	url:   &charm.URL{"cs", "", "foo", 1, ""},
}, {
	s:     "n0-n0-n0",
	exact: "cs:n0-n0-n0",
	url:   &charm.URL{"cs", "", "n0-n0-n0", -1, ""},
}, {
	s:     "cs:foo",
	exact: "cs:foo",
	url:   &charm.URL{"cs", "", "foo", -1, ""},
}, {
	s:     "local:foo",
	exact: "local:foo",
	url:   &charm.URL{"local", "", "foo", -1, ""},
}, {
	s:     "series/foo",
	exact: "cs:series/foo",
	url:   &charm.URL{"cs", "", "foo", -1, "series"},
}, {
	s:   "series/foo/bar",
	err: `charm or bundle URL has invalid form: "series/foo/bar"`,
}, {
	s:   "cs:foo/~blah",
	err: `cannot parse URL $URL: name "~blah" not valid`,
}}

func (s *URLSuite) TestParseURL(c *gc.C) {
	for i, t := range urlTests {
		c.Logf("test %d: %q", i, t.s)

		expectStr := t.s
		if t.exact != "" {
			expectStr = t.exact
		}
		url, uerr := charm.ParseURL(t.s)
		if t.err != "" {
			t.err = strings.Replace(t.err, "$URL", regexp.QuoteMeta(fmt.Sprintf("%q", t.s)), -1)
			c.Check(uerr, gc.ErrorMatches, t.err)
			c.Check(url, gc.IsNil)
			continue
		}
		c.Assert(uerr, gc.IsNil)
		c.Check(url, gc.DeepEquals, t.url)
		c.Check(url.String(), gc.Equals, expectStr)

		// URL strings are generated as expected.  Reversability is preserved
		// with v1 URLs.
		if t.exact != "" {
			c.Check(url.String(), gc.Equals, t.exact)
		} else {
			c.Check(url.String(), gc.Equals, t.s)
		}
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
	c.Assert(err, gc.ErrorMatches, "URL without charm or bundle name: .*")
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
			c.Assert(err, gc.ErrorMatches, fmt.Sprintf("cannot infer charm or bundle URL for %q: charm or bundle url series is not resolved", t.vague))
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
	c.Assert(f, gc.PanicMatches, "cannot parse URL \"local:@@/name\": series name \"@@\" not valid")
	f = func() { charm.MustParseURL("cs:~user") }
	c.Assert(f, gc.PanicMatches, "URL without charm or bundle name: .*")
	f = func() { charm.MustParseURL("cs:~user") }
	c.Assert(f, gc.PanicMatches, "URL without charm or bundle name: .*")
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
	Name      string
	Marshal   func(interface{}) ([]byte, error)
	Unmarshal func([]byte, interface{}) error
}{{
	Name:      "bson",
	Marshal:   bson.Marshal,
	Unmarshal: bson.Unmarshal,
}, {
	Name:      "json",
	Marshal:   json.Marshal,
	Unmarshal: json.Unmarshal,
}, {
	Name:      "yaml",
	Marshal:   yaml.Marshal,
	Unmarshal: yaml.Unmarshal,
}}

func (s *URLSuite) TestURLCodecs(c *gc.C) {
	for i, codec := range codecs {
		c.Logf("codec %d: %v", i, codec.Name)
		type doc struct {
			URL *charm.URL `json:",omitempty" bson:",omitempty" yaml:",omitempty"`
		}
		url := charm.MustParseURL("cs:series/name")
		v0 := doc{url}
		data, err := codec.Marshal(v0)
		c.Assert(err, gc.IsNil)
		var v doc
		err = codec.Unmarshal(data, &v)
		c.Assert(v, gc.DeepEquals, v0)

		// Check that the underlying representation
		// is a string.
		type strDoc struct {
			URL string
		}
		var vs strDoc
		err = codec.Unmarshal(data, &vs)
		c.Assert(err, gc.IsNil)
		c.Assert(vs.URL, gc.Equals, "cs:series/name")

		data, err = codec.Marshal(doc{})
		c.Assert(err, gc.IsNil)
		v = doc{}
		err = codec.Unmarshal(data, &v)
		c.Assert(err, gc.IsNil)
		c.Assert(v.URL, gc.IsNil, gc.Commentf("data: %q", data))
	}
}

func (s *URLSuite) TestJSONGarbage(c *gc.C) {
	// unmarshalling json gibberish
	for _, value := range []string{":{", `"cs:{}+<"`, `"cs:~_~/f00^^&^/baaaar$%-?"`} {
		err := json.Unmarshal([]byte(value), new(struct{ URL *charm.URL }))
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
