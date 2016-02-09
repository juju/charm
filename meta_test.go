// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/yaml.v1"

	"gopkg.in/juju/charm.v6-unstable"
	"gopkg.in/juju/charm.v6-unstable/resource"
)

func repoMeta(c *gc.C, name string) io.Reader {
	charmDir := charmDirPath(c, name)
	file, err := os.Open(filepath.Join(charmDir, "metadata.yaml"))
	c.Assert(err, gc.IsNil)
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	c.Assert(err, gc.IsNil)
	return bytes.NewReader(data)
}

type MetaSuite struct{}

var _ = gc.Suite(&MetaSuite{})

func (s *MetaSuite) TestReadMetaVersion1(c *gc.C) {
	meta, err := charm.ReadMeta(repoMeta(c, "dummy"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Name, gc.Equals, "dummy")
	c.Assert(meta.Summary, gc.Equals, "That's a dummy charm.")
	c.Assert(meta.Description, gc.Equals,
		"This is a longer description which\npotentially contains multiple lines.\n")
	c.Assert(meta.Format, gc.Equals, 1)
	c.Assert(meta.OldRevision, gc.Equals, 0)
	c.Assert(meta.Subordinate, gc.Equals, false)
}

func (s *MetaSuite) TestReadMetaVersion2(c *gc.C) {
	meta, err := charm.ReadMeta(repoMeta(c, "format2"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Name, gc.Equals, "format2")
	c.Assert(meta.Format, gc.Equals, 2)
	c.Assert(meta.Categories, gc.HasLen, 0)
	c.Assert(meta.Terms, gc.HasLen, 0)
}

func (s *MetaSuite) TestCheckTerms(c *gc.C) {
	tests := []struct {
		about       string
		terms       []string
		expectError string
	}{{
		about: "valid terms",
		terms: []string{"term/1", "term/2"},
	}, {
		about:       "missing revision number",
		terms:       []string{"term/1", "term"},
		expectError: "invalid term name \"term\": must match.*",
	}, {
		about:       "revision not a number",
		terms:       []string{"term/1", "term/a"},
		expectError: "invalid term name \"term/a\": must match.*",
	}, {
		about:       "wrong format",
		terms:       []string{"term/1", "term/a/1"},
		expectError: "invalid term name \"term/a/1\": must match.*",
	}, {
		about:       "term may not contain spaces",
		terms:       []string{"term/1", "term about a term"},
		expectError: "invalid term name \"term about a term\": must match.*",
	}, {
		about:       "term name must start with lowercase letter",
		terms:       []string{"Term/1"},
		expectError: `invalid term name "Term/1": must match.*`,
	}, {
		about:       "term name match the regexp",
		terms:       []string{"term_123-23aAf/1"},
		expectError: "invalid term name \"term_123-23aAf/1\": must match.*",
	},
	}
	for i, test := range tests {
		c.Logf("running test %v: %v", i, test.about)
		meta := charm.Meta{Terms: test.terms}
		err := meta.Check()
		if test.expectError == "" {
			c.Assert(err, jc.ErrorIsNil)
		} else {
			c.Assert(err, gc.ErrorMatches, test.expectError)
		}
	}
}

func (s *MetaSuite) TestReadCategory(c *gc.C) {
	meta, err := charm.ReadMeta(repoMeta(c, "category"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Categories, jc.DeepEquals, []string{"database"})
}

func (s *MetaSuite) TestReadTerms(c *gc.C) {
	meta, err := charm.ReadMeta(repoMeta(c, "terms"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Terms, jc.DeepEquals, []string{"term1", "term2"})
}

func (s *MetaSuite) TestReadTags(c *gc.C) {
	meta, err := charm.ReadMeta(repoMeta(c, "category"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Tags, jc.DeepEquals, []string{"openstack", "storage"})
}

func (s *MetaSuite) TestSubordinate(c *gc.C) {
	meta, err := charm.ReadMeta(repoMeta(c, "logging"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Subordinate, gc.Equals, true)
}

func (s *MetaSuite) TestSubordinateWithoutContainerRelation(c *gc.C) {
	r := repoMeta(c, "dummy")
	hackYaml := ReadYaml(r)
	hackYaml["subordinate"] = true
	_, err := charm.ReadMeta(hackYaml.Reader())
	c.Assert(err, gc.ErrorMatches, "subordinate charm \"dummy\" lacks \"requires\" relation with container scope")
}

func (s *MetaSuite) TestScopeConstraint(c *gc.C) {
	meta, err := charm.ReadMeta(repoMeta(c, "logging"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Provides["logging-client"].Scope, gc.Equals, charm.ScopeGlobal)
	c.Assert(meta.Requires["logging-directory"].Scope, gc.Equals, charm.ScopeContainer)
	c.Assert(meta.Subordinate, gc.Equals, true)
}

func (s *MetaSuite) TestParseMetaRelations(c *gc.C) {
	meta, err := charm.ReadMeta(repoMeta(c, "mysql"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Provides["server"], gc.Equals, charm.Relation{
		Name:      "server",
		Role:      charm.RoleProvider,
		Interface: "mysql",
		Scope:     charm.ScopeGlobal,
	})
	c.Assert(meta.Requires, gc.IsNil)
	c.Assert(meta.Peers, gc.IsNil)

	meta, err = charm.ReadMeta(repoMeta(c, "riak"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Provides["endpoint"], gc.Equals, charm.Relation{
		Name:      "endpoint",
		Role:      charm.RoleProvider,
		Interface: "http",
		Scope:     charm.ScopeGlobal,
	})
	c.Assert(meta.Provides["admin"], gc.Equals, charm.Relation{
		Name:      "admin",
		Role:      charm.RoleProvider,
		Interface: "http",
		Scope:     charm.ScopeGlobal,
	})
	c.Assert(meta.Peers["ring"], gc.Equals, charm.Relation{
		Name:      "ring",
		Role:      charm.RolePeer,
		Interface: "riak",
		Limit:     1,
		Scope:     charm.ScopeGlobal,
	})
	c.Assert(meta.Requires, gc.IsNil)

	meta, err = charm.ReadMeta(repoMeta(c, "terracotta"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Provides["dso"], gc.Equals, charm.Relation{
		Name:      "dso",
		Role:      charm.RoleProvider,
		Interface: "terracotta",
		Optional:  true,
		Scope:     charm.ScopeGlobal,
	})
	c.Assert(meta.Peers["server-array"], gc.Equals, charm.Relation{
		Name:      "server-array",
		Role:      charm.RolePeer,
		Interface: "terracotta-server",
		Limit:     1,
		Scope:     charm.ScopeGlobal,
	})
	c.Assert(meta.Requires, gc.IsNil)

	meta, err = charm.ReadMeta(repoMeta(c, "wordpress"))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Provides["url"], gc.Equals, charm.Relation{
		Name:      "url",
		Role:      charm.RoleProvider,
		Interface: "http",
		Scope:     charm.ScopeGlobal,
	})
	c.Assert(meta.Requires["db"], gc.Equals, charm.Relation{
		Name:      "db",
		Role:      charm.RoleRequirer,
		Interface: "mysql",
		Limit:     1,
		Scope:     charm.ScopeGlobal,
	})
	c.Assert(meta.Requires["cache"], gc.Equals, charm.Relation{
		Name:      "cache",
		Role:      charm.RoleRequirer,
		Interface: "varnish",
		Limit:     2,
		Optional:  true,
		Scope:     charm.ScopeGlobal,
	})
	c.Assert(meta.Peers, gc.IsNil)
}

var relationsConstraintsTests = []struct {
	rels string
	err  string
}{
	{
		"provides:\n  foo: ping\nrequires:\n  foo: pong",
		`charm "a" using a duplicated relation name: "foo"`,
	}, {
		"requires:\n  foo: ping\npeers:\n  foo: pong",
		`charm "a" using a duplicated relation name: "foo"`,
	}, {
		"peers:\n  foo: ping\nprovides:\n  foo: pong",
		`charm "a" using a duplicated relation name: "foo"`,
	}, {
		"provides:\n  juju: blob",
		`charm "a" using a reserved relation name: "juju"`,
	}, {
		"requires:\n  juju: blob",
		`charm "a" using a reserved relation name: "juju"`,
	}, {
		"peers:\n  juju: blob",
		`charm "a" using a reserved relation name: "juju"`,
	}, {
		"provides:\n  juju-snap: blub",
		`charm "a" using a reserved relation name: "juju-snap"`,
	}, {
		"requires:\n  juju-crackle: blub",
		`charm "a" using a reserved relation name: "juju-crackle"`,
	}, {
		"peers:\n  juju-pop: blub",
		`charm "a" using a reserved relation name: "juju-pop"`,
	}, {
		"provides:\n  innocuous: juju",
		`charm "a" relation "innocuous" using a reserved interface: "juju"`,
	}, {
		"peers:\n  innocuous: juju",
		`charm "a" relation "innocuous" using a reserved interface: "juju"`,
	}, {
		"provides:\n  innocuous: juju-snap",
		`charm "a" relation "innocuous" using a reserved interface: "juju-snap"`,
	}, {
		"peers:\n  innocuous: juju-snap",
		`charm "a" relation "innocuous" using a reserved interface: "juju-snap"`,
	},
}

func (s *MetaSuite) TestRelationsConstraints(c *gc.C) {
	check := func(s, e string) {
		meta, err := charm.ReadMeta(strings.NewReader(s))
		if e != "" {
			c.Assert(err, gc.ErrorMatches, e)
			c.Assert(meta, gc.IsNil)
		} else {
			c.Assert(err, gc.IsNil)
			c.Assert(meta, gc.NotNil)
		}
	}
	prefix := "name: a\nsummary: b\ndescription: c\n"
	for i, t := range relationsConstraintsTests {
		c.Logf("test %d", i)
		check(prefix+t.rels, t.err)
		check(prefix+"subordinate: true\n"+t.rels, t.err)
	}
	// The juju-* namespace is accessible to container-scoped require
	// relations on subordinate charms.
	check(prefix+`
subordinate: true
requires:
  juju-info:
    interface: juju-info
    scope: container`, "")
	// The juju-* interfaces are allowed on any require relation.
	check(prefix+`
requires:
  innocuous: juju-info`, "")
}

// dummyMetadata contains a minimally valid charm metadata.yaml
// for testing valid and invalid series.
const dummyMetadata = "name: a\nsummary: b\ndescription: c"

// TestSeries ensures that valid series values are parsed correctly when specified
// in the charm metadata.
func (s *MetaSuite) TestSeries(c *gc.C) {
	// series not specified
	meta, err := charm.ReadMeta(strings.NewReader(dummyMetadata))
	c.Assert(err, gc.IsNil)
	c.Check(meta.Series, gc.HasLen, 0)
	charmMeta := fmt.Sprintf("%s\nseries:", dummyMetadata)
	for _, seriesName := range []string{"precise", "trusty", "plan9"} {
		charmMeta = fmt.Sprintf("%s\n    - %s", charmMeta, seriesName)
	}
	meta, err = charm.ReadMeta(strings.NewReader(charmMeta))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Series, gc.DeepEquals, []string{"precise", "trusty", "plan9"})
}

// TestInvalidSeries ensures that invalid series values cause a parse error
// when specified in the charm metadata.
func (s *MetaSuite) TestInvalidSeries(c *gc.C) {
	for _, seriesName := range []string{"pre-c1se", "pre^cise", "cp/m", "OpenVMS"} {
		_, err := charm.ReadMeta(strings.NewReader(
			fmt.Sprintf("%s\nseries:\n    - %s\n", dummyMetadata, seriesName)))
		c.Assert(err, gc.NotNil)
		c.Check(err, gc.ErrorMatches, `charm "a" declares invalid series: .*`)
	}
}

func (s *MetaSuite) TestCheckMismatchedRelationName(c *gc.C) {
	// This  Check case cannot be covered by the above
	// TestRelationsConstraints tests.
	meta := charm.Meta{
		Name: "foo",
		Provides: map[string]charm.Relation{
			"foo": {
				Name:      "foo",
				Role:      charm.RolePeer,
				Interface: "x",
				Limit:     1,
				Scope:     charm.ScopeGlobal,
			},
		},
	}
	err := meta.Check()
	c.Assert(err, gc.ErrorMatches, `charm "foo" has mismatched role "peer"; expected "provider"`)
}

func (s *MetaSuite) TestCheckMismatchedRole(c *gc.C) {
	// This  Check case cannot be covered by the above
	// TestRelationsConstraints tests.
	meta := charm.Meta{
		Name: "foo",
		Provides: map[string]charm.Relation{
			"foo": {
				Role:      charm.RolePeer,
				Interface: "foo",
				Limit:     1,
				Scope:     charm.ScopeGlobal,
			},
		},
	}
	err := meta.Check()
	c.Assert(err, gc.ErrorMatches, `charm "foo" has mismatched relation name ""; expected "foo"`)
}

// Test rewriting of a given interface specification into long form.
//
// InterfaceExpander uses `coerce` to do one of two things:
//
//   - Rewrite shorthand to the long form used for actual storage
//   - Fills in defaults, including a configurable `limit`
//
// This test ensures test coverage on each of these branches, along
// with ensuring the conversion object properly raises SchemaError
// exceptions on invalid data.
func (s *MetaSuite) TestIfaceExpander(c *gc.C) {
	e := charm.IfaceExpander(nil)

	path := []string{"<pa", "th>"}

	// Shorthand is properly rewritten
	v, err := e.Coerce("http", path)
	c.Assert(err, gc.IsNil)
	c.Assert(v, jc.DeepEquals, map[string]interface{}{"interface": "http", "limit": nil, "optional": false, "scope": string(charm.ScopeGlobal)})

	// Defaults are properly applied
	v, err = e.Coerce(map[string]interface{}{"interface": "http"}, path)
	c.Assert(err, gc.IsNil)
	c.Assert(v, jc.DeepEquals, map[string]interface{}{"interface": "http", "limit": nil, "optional": false, "scope": string(charm.ScopeGlobal)})

	v, err = e.Coerce(map[string]interface{}{"interface": "http", "limit": 2}, path)
	c.Assert(err, gc.IsNil)
	c.Assert(v, jc.DeepEquals, map[string]interface{}{"interface": "http", "limit": int64(2), "optional": false, "scope": string(charm.ScopeGlobal)})

	v, err = e.Coerce(map[string]interface{}{"interface": "http", "optional": true}, path)
	c.Assert(err, gc.IsNil)
	c.Assert(v, jc.DeepEquals, map[string]interface{}{"interface": "http", "limit": nil, "optional": true, "scope": string(charm.ScopeGlobal)})

	// Invalid data raises an error.
	v, err = e.Coerce(42, path)
	c.Assert(err, gc.ErrorMatches, `<path>: expected map, got int\(42\)`)

	v, err = e.Coerce(map[string]interface{}{"interface": "http", "optional": nil}, path)
	c.Assert(err, gc.ErrorMatches, "<path>.optional: expected bool, got nothing")

	v, err = e.Coerce(map[string]interface{}{"interface": "http", "limit": "none, really"}, path)
	c.Assert(err, gc.ErrorMatches, "<path>.limit: unexpected value.*")

	// Can change default limit
	e = charm.IfaceExpander(1)
	v, err = e.Coerce(map[string]interface{}{"interface": "http"}, path)
	c.Assert(err, gc.IsNil)
	c.Assert(v, jc.DeepEquals, map[string]interface{}{"interface": "http", "limit": int64(1), "optional": false, "scope": string(charm.ScopeGlobal)})
}

func (s *MetaSuite) TestMetaHooks(c *gc.C) {
	meta, err := charm.ReadMeta(repoMeta(c, "wordpress"))
	c.Assert(err, gc.IsNil)
	hooks := meta.Hooks()
	expectedHooks := map[string]bool{
		"install":                           true,
		"start":                             true,
		"config-changed":                    true,
		"upgrade-charm":                     true,
		"stop":                              true,
		"collect-metrics":                   true,
		"meter-status-changed":              true,
		"leader-elected":                    true,
		"leader-deposed":                    true,
		"leader-settings-changed":           true,
		"update-status":                     true,
		"cache-relation-joined":             true,
		"cache-relation-changed":            true,
		"cache-relation-departed":           true,
		"cache-relation-broken":             true,
		"db-relation-joined":                true,
		"db-relation-changed":               true,
		"db-relation-departed":              true,
		"db-relation-broken":                true,
		"logging-dir-relation-joined":       true,
		"logging-dir-relation-changed":      true,
		"logging-dir-relation-departed":     true,
		"logging-dir-relation-broken":       true,
		"monitoring-port-relation-joined":   true,
		"monitoring-port-relation-changed":  true,
		"monitoring-port-relation-departed": true,
		"monitoring-port-relation-broken":   true,
		"url-relation-joined":               true,
		"url-relation-changed":              true,
		"url-relation-departed":             true,
		"url-relation-broken":               true,
	}
	c.Assert(hooks, jc.DeepEquals, expectedHooks)
}

func (s *MetaSuite) TestCodecRoundTripEmpty(c *gc.C) {
	for i, codec := range codecs {
		c.Logf("codec %d", i)
		empty_input := charm.Meta{}
		data, err := codec.Marshal(empty_input)
		c.Assert(err, gc.IsNil)
		var empty_output charm.Meta
		err = codec.Unmarshal(data, &empty_output)
		c.Assert(err, gc.IsNil)
		c.Assert(empty_input, jc.DeepEquals, empty_output)
	}
}

func (s *MetaSuite) TestCodecRoundTrip(c *gc.C) {
	var input = charm.Meta{
		Name:        "Foo",
		Summary:     "Bar",
		Description: "Baz",
		Subordinate: true,
		Provides: map[string]charm.Relation{
			"qux": {
				Interface: "quxx",
				Optional:  true,
				Limit:     42,
				Scope:     "quxxx",
			},
		},
		Requires: map[string]charm.Relation{
			"qux": {
				Interface: "quxx",
				Optional:  true,
				Limit:     42,
				Scope:     "quxxx",
			},
		},
		Peers: map[string]charm.Relation{
			"qux": {
				Interface: "quxx",
				Optional:  true,
				Limit:     42,
				Scope:     "quxxx",
			},
		},
		Categories:  []string{"quxxxx", "quxxxxx"},
		Tags:        []string{"openstack", "storage"},
		Format:      10,
		OldRevision: 11,
		Terms:       []string{"test term 1", "test term 2"},
	}
	for i, codec := range codecs {
		c.Logf("codec %d", i)
		data, err := codec.Marshal(input)
		c.Assert(err, gc.IsNil)
		var output charm.Meta
		err = codec.Unmarshal(data, &output)
		c.Assert(err, gc.IsNil)
		c.Assert(input, jc.DeepEquals, output)
	}
}

var implementedByTests = []struct {
	ifce     string
	name     string
	role     charm.RelationRole
	scope    charm.RelationScope
	match    bool
	implicit bool
}{
	{"ifce-pro", "pro", charm.RoleProvider, charm.ScopeGlobal, true, false},
	{"blah", "pro", charm.RoleProvider, charm.ScopeGlobal, false, false},
	{"ifce-pro", "blah", charm.RoleProvider, charm.ScopeGlobal, false, false},
	{"ifce-pro", "pro", charm.RoleRequirer, charm.ScopeGlobal, false, false},
	{"ifce-pro", "pro", charm.RoleProvider, charm.ScopeContainer, true, false},

	{"juju-info", "juju-info", charm.RoleProvider, charm.ScopeGlobal, true, true},
	{"blah", "juju-info", charm.RoleProvider, charm.ScopeGlobal, false, false},
	{"juju-info", "blah", charm.RoleProvider, charm.ScopeGlobal, false, false},
	{"juju-info", "juju-info", charm.RoleRequirer, charm.ScopeGlobal, false, false},
	{"juju-info", "juju-info", charm.RoleProvider, charm.ScopeContainer, true, true},

	{"ifce-req", "req", charm.RoleRequirer, charm.ScopeGlobal, true, false},
	{"blah", "req", charm.RoleRequirer, charm.ScopeGlobal, false, false},
	{"ifce-req", "blah", charm.RoleRequirer, charm.ScopeGlobal, false, false},
	{"ifce-req", "req", charm.RolePeer, charm.ScopeGlobal, false, false},
	{"ifce-req", "req", charm.RoleRequirer, charm.ScopeContainer, true, false},

	{"juju-info", "info", charm.RoleRequirer, charm.ScopeContainer, true, false},
	{"blah", "info", charm.RoleRequirer, charm.ScopeContainer, false, false},
	{"juju-info", "blah", charm.RoleRequirer, charm.ScopeContainer, false, false},
	{"juju-info", "info", charm.RolePeer, charm.ScopeContainer, false, false},
	{"juju-info", "info", charm.RoleRequirer, charm.ScopeGlobal, false, false},

	{"ifce-peer", "peer", charm.RolePeer, charm.ScopeGlobal, true, false},
	{"blah", "peer", charm.RolePeer, charm.ScopeGlobal, false, false},
	{"ifce-peer", "blah", charm.RolePeer, charm.ScopeGlobal, false, false},
	{"ifce-peer", "peer", charm.RoleProvider, charm.ScopeGlobal, false, false},
	{"ifce-peer", "peer", charm.RolePeer, charm.ScopeContainer, true, false},
}

func (s *MetaSuite) TestImplementedBy(c *gc.C) {
	for i, t := range implementedByTests {
		c.Logf("test %d", i)
		r := charm.Relation{
			Interface: t.ifce,
			Name:      t.name,
			Role:      t.role,
			Scope:     t.scope,
		}
		c.Assert(r.ImplementedBy(&dummyCharm{}), gc.Equals, t.match)
		c.Assert(r.IsImplicit(), gc.Equals, t.implicit)
	}
}

var metaYAMLMarshalTests = []struct {
	about string
	yaml  string
}{{
	about: "minimal charm",
	yaml: `
name: minimal
description: d
summary: s
`,
}, {
	about: "charm with lots of stuff",
	yaml: `
name: big
description: d
summary: s
subordinate: true
provides:
    provideSimple: someinterface
    provideLessSimple:
        interface: anotherinterface
        optional: true
        scope: container
        limit: 3
requires:
    requireSimple: someinterface
    requireLessSimple:
        interface: anotherinterface
        optional: true
        scope: container
        limit: 3
peers:
    peerSimple: someinterface
    peerLessSimple:
        interface: peery
        optional: true
categories: [c1, c1]
tags: [t1, t2]
series:
    - someseries
`,
}}

func (s *MetaSuite) TestYAMLMarshal(c *gc.C) {
	for i, test := range metaYAMLMarshalTests {
		c.Logf("test %d: %s", i, test.about)
		ch, err := charm.ReadMeta(strings.NewReader(test.yaml))
		c.Assert(err, gc.IsNil)
		gotYAML, err := yaml.Marshal(ch)
		c.Assert(err, gc.IsNil)
		gotCh, err := charm.ReadMeta(bytes.NewReader(gotYAML))
		c.Assert(err, gc.IsNil)
		c.Assert(gotCh, jc.DeepEquals, ch)
	}
}

func (s *MetaSuite) TestYAMLMarshalSimpleRelation(c *gc.C) {
	// Check that a simple relation gets marshaled as a string.
	chYAML := `
name: minimal
description: d
summary: s
provides:
    server: http
requires:
    client: http
peers:
     me: http
`
	ch, err := charm.ReadMeta(strings.NewReader(chYAML))
	c.Assert(err, gc.IsNil)
	gotYAML, err := yaml.Marshal(ch)
	c.Assert(err, gc.IsNil)

	var x interface{}
	err = yaml.Unmarshal(gotYAML, &x)
	c.Assert(err, gc.IsNil)
	c.Assert(x, jc.DeepEquals, map[interface{}]interface{}{
		"name":        "minimal",
		"description": "d",
		"summary":     "s",
		"provides": map[interface{}]interface{}{
			"server": "http",
		},
		"requires": map[interface{}]interface{}{
			"client": "http",
		},
		"peers": map[interface{}]interface{}{
			"me": "http",
		},
	})
}

func (s *MetaSuite) TestStorage(c *gc.C) {
	// "type" is the only required attribute for storage.
	meta, err := charm.ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
storage:
    store0:
        description: woo tee bix
        type: block
    store1:
        type: filesystem
`))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Storage, gc.DeepEquals, map[string]charm.Storage{
		"store0": {
			Name:        "store0",
			Description: "woo tee bix",
			Type:        charm.StorageBlock,
			CountMin:    1, // singleton
			CountMax:    1,
		},
		"store1": {
			Name:     "store1",
			Type:     charm.StorageFilesystem,
			CountMin: 1, // singleton
			CountMax: 1,
		},
	})
}

func (s *MetaSuite) TestStorageErrors(c *gc.C) {
	prefix := `
name: a
summary: b
description: c
storage:
 store-bad:
`[1:]

	type test struct {
		desc string
		yaml string
		err  string
	}

	tests := []test{{
		desc: "type is required",
		yaml: "  required: false",
		err:  "metadata: storage.store-bad.type: unexpected value <nil>",
	}, {
		desc: "range must be an integer, or integer range (1)",
		yaml: "  type: filesystem\n  multiple:\n   range: woat",
		err:  `metadata: storage.store-bad.multiple.range: value "woat" does not match 'm', 'm-n', or 'm\+'`,
	}, {
		desc: "range must be an integer, or integer range (2)",
		yaml: "  type: filesystem\n  multiple:\n   range: 0-abc",
		err:  `metadata: storage.store-bad.multiple.range: value "0-abc" does not match 'm', 'm-n', or 'm\+'`,
	}, {
		desc: "range must be non-negative",
		yaml: "  type: filesystem\n  multiple:\n    range: -1",
		err:  `metadata: storage.store-bad.multiple.range: invalid count -1`,
	}, {
		desc: "range must be positive",
		yaml: "  type: filesystem\n  multiple:\n    range: 0",
		err:  `metadata: storage.store-bad.multiple.range: invalid count 0`,
	}, {
		desc: "location cannot be specified for block type storage",
		yaml: "  type: block\n  location: /dev/sdc",
		err:  `charm "a" storage "store-bad": location may not be specified for "type: block"`,
	}, {
		desc: "minimum size must parse correctly",
		yaml: "  type: block\n  minimum-size: foo",
		err:  `metadata: expected a non-negative number, got "foo"`,
	}, {
		desc: "minimum size must have valid suffix",
		yaml: "  type: block\n  minimum-size: 10Q",
		err:  `metadata: invalid multiplier suffix "Q", expected one of MGTPEZY`,
	}, {
		desc: "properties must contain valid values",
		yaml: "  type: block\n  properties: [transient, foo]",
		err:  `metadata: .* unexpected value "foo"`,
	}}

	for i, test := range tests {
		c.Logf("test %d: %s", i, test.desc)
		c.Logf("\n%s\n", prefix+test.yaml)
		_, err := charm.ReadMeta(strings.NewReader(prefix + test.yaml))
		c.Assert(err, gc.ErrorMatches, test.err)
	}
}

func (s *MetaSuite) TestStorageCount(c *gc.C) {
	testStorageCount := func(count string, min, max int) {
		meta, err := charm.ReadMeta(strings.NewReader(fmt.Sprintf(`
name: a
summary: b
description: c
storage:
    store0:
        type: filesystem
        multiple:
            range: %s
`, count)))
		c.Assert(err, gc.IsNil)
		store := meta.Storage["store0"]
		c.Assert(store, gc.NotNil)
		c.Assert(store.CountMin, gc.Equals, min)
		c.Assert(store.CountMax, gc.Equals, max)
	}
	testStorageCount("1", 1, 1)
	testStorageCount("0-1", 0, 1)
	testStorageCount("1-1", 1, 1)
	testStorageCount("1+", 1, -1)
	// n- is equivalent to n+
	testStorageCount("1-", 1, -1)
}

func (s *MetaSuite) TestStorageLocation(c *gc.C) {
	meta, err := charm.ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
storage:
    store0:
        type: filesystem
        location: /var/lib/things
`))
	c.Assert(err, gc.IsNil)
	store := meta.Storage["store0"]
	c.Assert(store, gc.NotNil)
	c.Assert(store.Location, gc.Equals, "/var/lib/things")
}

func (s *MetaSuite) TestStorageMinimumSize(c *gc.C) {
	meta, err := charm.ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
storage:
    store0:
        type: filesystem
        minimum-size: 10G
`))
	c.Assert(err, gc.IsNil)
	store := meta.Storage["store0"]
	c.Assert(store, gc.NotNil)
	c.Assert(store.MinimumSize, gc.Equals, uint64(10*1024))
}

func (s *MetaSuite) TestStorageProperties(c *gc.C) {
	meta, err := charm.ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
storage:
    store0:
        type: filesystem
        properties: [transient]
`))
	c.Assert(err, gc.IsNil)
	store := meta.Storage["store0"]
	c.Assert(store, gc.NotNil)
	c.Assert(store.Properties, jc.SameContents, []string{"transient"})
}

func (s *MetaSuite) TestPayloadClasses(c *gc.C) {
	meta, err := charm.ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
payloads:
    monitor:
        type: docker
    kvm-guest:
        type: kvm
`))
	c.Assert(err, gc.IsNil)

	c.Check(meta.PayloadClasses, jc.DeepEquals, map[string]charm.PayloadClass{
		"monitor": charm.PayloadClass{
			Name: "monitor",
			Type: "docker",
		},
		"kvm-guest": charm.PayloadClass{
			Name: "kvm-guest",
			Type: "kvm",
		},
	})
}

func (s *MetaSuite) TestResources(c *gc.C) {
	meta, err := charm.ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
resources:
    resource-name:
        type: file
        filename: filename.tgz
        description: "One line that is useful when operators need to push it."
    other-resource:
        type: file
        filename: other.zip
`))
	c.Assert(err, gc.IsNil)

	c.Check(meta.Resources, jc.DeepEquals, map[string]resource.Meta{
		"resource-name": resource.Meta{
			Name:        "resource-name",
			Type:        resource.TypeFile,
			Path:        "filename.tgz",
			Description: "One line that is useful when operators need to push it.",
		},
		"other-resource": resource.Meta{
			Name: "other-resource",
			Type: resource.TypeFile,
			Path: "other.zip",
		},
	})
}

type dummyCharm struct{}

func (c *dummyCharm) Config() *charm.Config {
	panic("unused")
}

func (c *dummyCharm) Metrics() *charm.Metrics {
	panic("unused")
}

func (c *dummyCharm) Actions() *charm.Actions {
	panic("unused")
}

func (c *dummyCharm) Revision() int {
	panic("unused")
}

func (c *dummyCharm) Meta() *charm.Meta {
	return &charm.Meta{
		Provides: map[string]charm.Relation{
			"pro": {Interface: "ifce-pro", Scope: charm.ScopeGlobal},
		},
		Requires: map[string]charm.Relation{
			"req":  {Interface: "ifce-req", Scope: charm.ScopeGlobal},
			"info": {Interface: "juju-info", Scope: charm.ScopeContainer},
		},
		Peers: map[string]charm.Relation{
			"peer": {Interface: "ifce-peer", Scope: charm.ScopeGlobal},
		},
	}
}
