// Copyright 2014 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/mgo.v2/bson"

	"gopkg.in/juju/charm.v7-unstable"
)

type bundleDataSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&bundleDataSuite{})

const mediawikiBundle = `
series: precise
applications:
    mediawiki:
        charm: "cs:precise/mediawiki-10"
        num_units: 1
        expose: true
        options:
            debug: false
            name: Please set name of wiki
            skin: vector
        annotations:
            "gui-x": 609
            "gui-y": -15
        storage:
            valid-store: 10G
        bindings:
            db: db
            website: public
        resources:
            data: 3
    mysql:
        charm: "cs:precise/mysql-28"
        num_units: 2
        to: [0, mediawiki/0]
        options:
            "binlog-format": MIXED
            "block-size": 5.3
            "dataset-size": "80%"
            flavor: distro
            "ha-bindiface": eth0
            "ha-mcastport": 5411.1
        annotations:
            "gui-x": 610
            "gui-y": 255
        constraints: "mem=8g"
        bindings:
            db: db
        resources:
            data: "resources/data.tar"
relations:
    - ["mediawiki:db", "mysql:db"]
    - ["mysql:foo", "mediawiki:bar"]
machines:
    0:
         constraints: 'arch=amd64 mem=4g'
         annotations:
             foo: bar
tags:
    - super
    - awesome
description: |
    Everything is awesome. Everything is cool when we work as a team.
    Lovely day.
`

var parseTests = []struct {
	about                         string
	data                          string
	expectedBD                    *charm.BundleData
	expectedErr                   string
	expectUnmarshaledWithServices bool
}{{
	about: "mediawiki",
	data:  mediawikiBundle,
	expectedBD: &charm.BundleData{
		Series: "precise",
		Applications: map[string]*charm.ApplicationSpec{
			"mediawiki": {
				Charm:    "cs:precise/mediawiki-10",
				NumUnits: 1,
				Expose:   true,
				Options: map[string]interface{}{
					"debug": false,
					"name":  "Please set name of wiki",
					"skin":  "vector",
				},
				Annotations: map[string]string{
					"gui-x": "609",
					"gui-y": "-15",
				},
				Storage: map[string]string{
					"valid-store": "10G",
				},
				EndpointBindings: map[string]string{
					"db":      "db",
					"website": "public",
				},
				Resources: map[string]interface{}{
					"data": 3,
				},
			},
			"mysql": {
				Charm:    "cs:precise/mysql-28",
				NumUnits: 2,
				To:       []string{"0", "mediawiki/0"},
				Options: map[string]interface{}{
					"binlog-format": "MIXED",
					"block-size":    5.3,
					"dataset-size":  "80%",
					"flavor":        "distro",
					"ha-bindiface":  "eth0",
					"ha-mcastport":  5411.1,
				},
				Annotations: map[string]string{
					"gui-x": "610",
					"gui-y": "255",
				},
				Constraints: "mem=8g",
				EndpointBindings: map[string]string{
					"db": "db",
				},
				Resources: map[string]interface{}{"data": "resources/data.tar"},
			},
		},
		Machines: map[string]*charm.MachineSpec{
			"0": {
				Constraints: "arch=amd64 mem=4g",
				Annotations: map[string]string{
					"foo": "bar",
				},
			},
		},
		Relations: [][]string{
			{"mediawiki:db", "mysql:db"},
			{"mysql:foo", "mediawiki:bar"},
		},
		Tags: []string{"super", "awesome"},
		Description: `Everything is awesome. Everything is cool when we work as a team.
Lovely day.
`,
	},
}, {
	about: "relations specified with hyphens",
	data: `
relations:
    - - "mediawiki:db"
      - "mysql:db"
    - - "mysql:foo"
      - "mediawiki:bar"
`,
	expectedBD: &charm.BundleData{
		Relations: [][]string{
			{"mediawiki:db", "mysql:db"},
			{"mysql:foo", "mediawiki:bar"},
		},
	},
}, {
	about: "legacy bundle with services instead of applications",
	data: `
services:
    wordpress:
        charm: wordpress
    mysql:
        charm: mysql
        num_units: 1
relations:
    - ["wordpress:db", "mysql:db"]
`,
	expectedBD: &charm.BundleData{
		Applications: map[string]*charm.ApplicationSpec{
			"wordpress": {
				Charm: "wordpress",
			},
			"mysql": {
				Charm:    "mysql",
				NumUnits: 1,
			},
		},
		Relations: [][]string{
			{"wordpress:db", "mysql:db"},
		},
	},
	expectUnmarshaledWithServices: true,
}, {
	about: "bundle with services and applications",
	data: `
applications:
    wordpress:
        charm: wordpress
services:
    wordpress:
        charm: wordpress
    mysql:
        charm: mysql
        num_units: 1
relations:
    - ["wordpress:db", "mysql:db"]
`,
	expectedErr: ".*cannot specify both applications and services",
}}

func (*bundleDataSuite) TestParse(c *gc.C) {
	for i, test := range parseTests {
		c.Logf("test %d: %s", i, test.about)
		bd, err := charm.ReadBundleData(strings.NewReader(test.data))
		if test.expectedErr != "" {
			c.Assert(err, gc.ErrorMatches, test.expectedErr)
			continue
		}
		c.Assert(err, gc.IsNil)
		c.Assert(bd.UnmarshaledWithServices(), gc.Equals, test.expectUnmarshaledWithServices)
		bd.ClearUnmarshaledWithServices()
		c.Assert(bd, jc.DeepEquals, test.expectedBD)
	}
}

func (*bundleDataSuite) TestCodecRoundTrip(c *gc.C) {
	for _, test := range parseTests {
		if test.expectedErr != "" {
			continue
		}
		// Check that for all the known codecs, we can
		// round-trip the bundle data through them.
		for _, codec := range codecs {
			data, err := codec.Marshal(test.expectedBD)
			c.Assert(err, gc.IsNil)
			var bd charm.BundleData
			err = codec.Unmarshal(data, &bd)
			c.Assert(err, gc.IsNil)

			for _, app := range bd.Applications {
				for resName, res := range app.Resources {
					if val, ok := res.(float64); ok {
						app.Resources[resName] = int(val)
					}
				}
			}

			c.Assert(&bd, jc.DeepEquals, test.expectedBD)
		}
	}
}

func (*bundleDataSuite) TestParseLocalWithSeries(c *gc.C) {
	path := "internal/test-charm-repo/quanta/riak"
	data := fmt.Sprintf(`
        applications:
            dummy:
                charm: %s
                series: xenial
                num_units: 1
    `, path)
	bd, err := charm.ReadBundleData(strings.NewReader(data))
	c.Assert(err, gc.IsNil)
	c.Assert(bd, jc.DeepEquals, &charm.BundleData{
		Applications: map[string]*charm.ApplicationSpec{
			"dummy": {
				Charm:    path,
				Series:   "xenial",
				NumUnits: 1,
			},
		}})
}

func (s *bundleDataSuite) TestUnmarshalWithServices(c *gc.C) {
	obj := map[string]interface{}{
		"services": map[string]interface{}{
			"wordpress": map[string]interface{}{
				"charm": "wordpress",
			},
		},
	}
	for i, codec := range codecs {
		c.Logf("codec %d: %v", i, codec.Name)
		data, err := codec.Marshal(obj)
		c.Assert(err, gc.IsNil)
		var bd charm.BundleData
		err = codec.Unmarshal(data, &bd)
		c.Assert(err, gc.IsNil)
		c.Assert(bd.UnmarshaledWithServices(), gc.Equals, true)
		bd.ClearUnmarshaledWithServices()
		c.Assert(bd, jc.DeepEquals, charm.BundleData{
			Applications: map[string]*charm.ApplicationSpec{"wordpress": {Charm: "wordpress"}}},
		)
	}
}

func (s *bundleDataSuite) TestBSONNilData(c *gc.C) {
	bd := map[string]*charm.BundleData{
		"test": nil,
	}
	data, err := bson.Marshal(bd)
	c.Assert(err, jc.ErrorIsNil)
	var result map[string]*charm.BundleData
	err = bson.Unmarshal(data, &result)
	c.Assert(err, gc.IsNil)
	c.Assert(result["test"], gc.IsNil)
}

var verifyErrorsTests = []struct {
	about  string
	data   string
	errors []string
}{{
	about: "as many errors as possible",
	data: `
series: "9wrong"

machines:
    0:
         constraints: 'bad constraints'
         annotations:
             foo: bar
         series: 'bad series'
    bogus:
    3:
applications:
    mediawiki:
        charm: "bogus:precise/mediawiki-10"
        num_units: -4
        options:
            debug: false
            name: Please set name of wiki
            skin: vector
        annotations:
            "gui-x": 609
            "gui-y": -15
        resources:
            "": 42
            "foo":
               "not": int
    riak:
        charm: "./somepath"
    mysql:
        charm: "cs:precise/mysql-28"
        num_units: 2
        to: [0, mediawiki/0, nowhere/3, 2, "bad placement"]
        options:
            "binlog-format": MIXED
            "block-size": 5
            "dataset-size": "80%"
            flavor: distro
            "ha-bindiface": eth0
            "ha-mcastport": 5411
        annotations:
            "gui-x": 610
            "gui-y": 255
        constraints: "bad constraints"
    wordpress:
          charm: wordpress
    postgres:
        charm: "cs:xenial/postgres"
        series: trusty
    terracotta:
        charm: "cs:xenial/terracotta"
        series: xenial
    ceph:
          charm: ceph
          storage:
              valid-storage: 3,10G
              no_underscores: 123
    ceph-osd:
          charm: ceph-osd
          storage:
              invalid-storage: "bad storage constraints"
relations:
    - ["mediawiki:db", "mysql:db"]
    - ["mysql:foo", "mediawiki:bar"]
    - ["arble:bar"]
    - ["arble:bar", "mediawiki:db"]
    - ["mysql:foo", "mysql:bar"]
    - ["mysql:db", "mediawiki:db"]
    - ["mediawiki/db", "mysql:db"]
    - ["wordpress", "mysql"]
`,
	errors: []string{
		`bundle declares an invalid series "9wrong"`,
		`invalid storage name "no_underscores" in application "ceph"`,
		`invalid storage "invalid-storage" in application "ceph-osd": bad storage constraint`,
		`machine "3" is not referred to by a placement directive`,
		`machine "bogus" is not referred to by a placement directive`,
		`invalid machine id "bogus" found in machines`,
		`invalid constraints "bad constraints" in machine "0": bad constraint`,
		`invalid charm URL in application "mediawiki": cannot parse URL "bogus:precise/mediawiki-10": schema "bogus" not valid`,
		`charm path in application "riak" does not exist: internal/test-charm-repo/bundle/somepath`,
		`invalid constraints "bad constraints" in application "mysql": bad constraint`,
		`negative number of units specified on application "mediawiki"`,
		`missing resource name on application "mediawiki"`,
		`resource revision "mediawiki" is not int or string`,
		`the charm URL for application "postgres" has a series which does not match, please remove the series from the URL`,
		`too many units specified in unit placement for application "mysql"`,
		`placement "nowhere/3" refers to an application not defined in this bundle`,
		`placement "mediawiki/0" specifies a unit greater than the -4 unit(s) started by the target application`,
		`placement "2" refers to a machine not defined in this bundle`,
		`relation ["arble:bar"] has 1 endpoint(s), not 2`,
		`relation ["arble:bar" "mediawiki:db"] refers to application "arble" not defined in this bundle`,
		`relation ["mysql:foo" "mysql:bar"] relates an application to itself`,
		`relation ["mysql:db" "mediawiki:db"] is defined more than once`,
		`invalid placement syntax "bad placement"`,
		`invalid relation syntax "mediawiki/db"`,
		`invalid series bad series for machine "0"`,
	},
}, {
	about: "mediawiki should be ok",
	data:  mediawikiBundle,
}}

func (*bundleDataSuite) TestVerifyErrors(c *gc.C) {
	for i, test := range verifyErrorsTests {
		c.Logf("test %d: %s", i, test.about)
		assertVerifyErrors(c, test.data, nil, test.errors)
	}
}

func assertVerifyErrors(c *gc.C, bundleData string, charms map[string]charm.Charm, expectErrors []string) {
	bd, err := charm.ReadBundleData(strings.NewReader(bundleData))
	c.Assert(err, gc.IsNil)

	validateConstraints := func(c string) error {
		if c == "bad constraints" {
			return fmt.Errorf("bad constraint")
		}
		return nil
	}
	validateStorage := func(c string) error {
		if c == "bad storage constraints" {
			return fmt.Errorf("bad storage constraint")
		}
		return nil
	}
	validateDevices := func(c string) error {
		if c == "bad device constraints" {
			return fmt.Errorf("bad device constraint")
		}
		return nil
	}
	if charms != nil {
		err = bd.VerifyWithCharms(validateConstraints, validateStorage, validateDevices, charms)
	} else {
		err = bd.VerifyLocal("internal/test-charm-repo/bundle", validateConstraints, validateStorage, validateDevices)
	}

	if len(expectErrors) == 0 {
		if err == nil {
			return
		}
		// Let the rest of the function deal with the
		// error, so that we'll see the actual errors
		// that resulted.
	}
	c.Assert(err, gc.FitsTypeOf, (*charm.VerificationError)(nil))
	errors := err.(*charm.VerificationError).Errors
	errStrings := make([]string, len(errors))
	for i, err := range errors {
		errStrings[i] = err.Error()
	}
	sort.Strings(errStrings)
	sort.Strings(expectErrors)
	c.Assert(errStrings, jc.DeepEquals, expectErrors)
}

func (*bundleDataSuite) TestVerifyCharmURL(c *gc.C) {
	bd, err := charm.ReadBundleData(strings.NewReader(mediawikiBundle))
	c.Assert(err, gc.IsNil)
	for i, u := range []string{
		"wordpress",
		"cs:wordpress",
		"cs:precise/wordpress",
		"precise/wordpress",
		"precise/wordpress-2",
		"local:foo",
		"local:foo-45",
	} {
		c.Logf("test %d: %s", i, u)
		bd.Applications["mediawiki"].Charm = u
		err := bd.Verify(nil, nil, nil)
		c.Check(err, gc.IsNil, gc.Commentf("charm url %q", u))
	}
}

func (*bundleDataSuite) TestVerifyLocalCharm(c *gc.C) {
	bd, err := charm.ReadBundleData(strings.NewReader(mediawikiBundle))
	c.Assert(err, gc.IsNil)
	bundleDir := c.MkDir()
	relativeCharmDir := filepath.Join(bundleDir, "charm")
	err = os.MkdirAll(relativeCharmDir, 0700)
	c.Assert(err, jc.ErrorIsNil)
	for i, u := range []string{
		"wordpress",
		"cs:wordpress",
		"cs:precise/wordpress",
		"precise/wordpress",
		"precise/wordpress-2",
		"local:foo",
		"local:foo-45",
		c.MkDir(),
		"./charm",
	} {
		c.Logf("test %d: %s", i, u)
		bd.Applications["mediawiki"].Charm = u
		err := bd.VerifyLocal(bundleDir, nil, nil, nil)
		c.Check(err, gc.IsNil, gc.Commentf("charm url %q", u))
	}
}

func (s *bundleDataSuite) TestVerifyBundleUsingJujuInfoRelation(c *gc.C) {
	err := s.testPrepareAndMutateBeforeVerifyWithCharms(c, nil)
	c.Assert(err, gc.IsNil)
}

func (s *bundleDataSuite) testPrepareAndMutateBeforeVerifyWithCharms(c *gc.C, mutator func(bd *charm.BundleData)) error {
	b := readBundleDir(c, "wordpress-with-logging")
	bd := b.Data()

	charms := map[string]charm.Charm{
		"wordpress": readCharmDir(c, "wordpress"),
		"mysql":     readCharmDir(c, "mysql"),
		"logging":   readCharmDir(c, "logging"),
	}

	if mutator != nil {
		mutator(bd)
	}

	return bd.VerifyWithCharms(nil, nil, nil, charms)
}

func (s *bundleDataSuite) TestVerifyBundleWithUnknownEndpointBindingGiven(c *gc.C) {
	err := s.testPrepareAndMutateBeforeVerifyWithCharms(c, func(bd *charm.BundleData) {
		bd.Applications["wordpress"].EndpointBindings["foo"] = "bar"
	})
	c.Assert(err, gc.ErrorMatches,
		`application "wordpress" wants to bind endpoint "foo" to space "bar", `+
			`but the endpoint is not defined by the charm`,
	)
}

func (s *bundleDataSuite) TestVerifyBundleWithExtraBindingsSuccess(c *gc.C) {
	err := s.testPrepareAndMutateBeforeVerifyWithCharms(c, func(bd *charm.BundleData) {
		// Both of these are specified in extra-bindings.
		bd.Applications["wordpress"].EndpointBindings["admin-api"] = "internal"
		bd.Applications["wordpress"].EndpointBindings["foo-bar"] = "test"
	})
	c.Assert(err, gc.IsNil)
}

func (s *bundleDataSuite) TestVerifyBundleWithRelationNameBindingSuccess(c *gc.C) {
	err := s.testPrepareAndMutateBeforeVerifyWithCharms(c, func(bd *charm.BundleData) {
		// Both of these are specified in as relations.
		bd.Applications["wordpress"].EndpointBindings["cache"] = "foo"
		bd.Applications["wordpress"].EndpointBindings["monitoring-port"] = "bar"
	})
	c.Assert(err, gc.IsNil)
}

func (*bundleDataSuite) TestRequiredCharms(c *gc.C) {
	bd, err := charm.ReadBundleData(strings.NewReader(mediawikiBundle))
	c.Assert(err, gc.IsNil)
	reqCharms := bd.RequiredCharms()

	c.Assert(reqCharms, gc.DeepEquals, []string{"cs:precise/mediawiki-10", "cs:precise/mysql-28"})
}

// testCharm returns a charm with the given name
// and relations. The relations are specified as
// a string of the form:
//
//	<provides-relations> | <requires-relations>
//
// Within each section, each white-space separated
// relation is specified as:
///	<relation-name>:<interface>
//
// So, for example:
//
//     testCharm("wordpress", "web:http | db:mysql")
//
// is equivalent to a charm with metadata.yaml containing
//
//	name: wordpress
//	description: wordpress
//	provides:
//	    web:
//	        interface: http
//	requires:
//	    db:
//	        interface: mysql
//
// If the charm name has a "-sub" suffix, the
// returned charm will have Meta.Subordinate = true.
//
func testCharm(name string, relations string) charm.Charm {
	var provides, requires string
	parts := strings.Split(relations, "|")
	provides = parts[0]
	if len(parts) > 1 {
		requires = parts[1]
	}
	meta := &charm.Meta{
		Name:        name,
		Summary:     name,
		Description: name,
		Provides:    parseRelations(provides, charm.RoleProvider),
		Requires:    parseRelations(requires, charm.RoleRequirer),
	}
	if strings.HasSuffix(name, "-sub") {
		meta.Subordinate = true
	}
	configStr := `
options:
  title: {default: My Title, description: title, type: string}
  skill-level: {description: skill, type: int}
`
	config, err := charm.ReadConfig(strings.NewReader(configStr))
	if err != nil {
		panic(err)
	}
	return testCharmImpl{
		meta:   meta,
		config: config,
	}
}

func parseRelations(s string, role charm.RelationRole) map[string]charm.Relation {
	rels := make(map[string]charm.Relation)
	for _, r := range strings.Fields(s) {
		parts := strings.Split(r, ":")
		if len(parts) != 2 {
			panic(fmt.Errorf("invalid relation specifier %q", r))
		}
		name, interf := parts[0], parts[1]
		rels[name] = charm.Relation{
			Name:      name,
			Role:      role,
			Interface: interf,
			Scope:     charm.ScopeGlobal,
		}
	}
	return rels
}

type testCharmImpl struct {
	meta   *charm.Meta
	config *charm.Config
	// Implement charm.Charm, but panic if anything other than
	// Meta or Config methods are called.
	charm.Charm
}

func (c testCharmImpl) Meta() *charm.Meta {
	return c.meta
}

func (c testCharmImpl) Config() *charm.Config {
	return c.config
}

var verifyWithCharmsErrorsTests = []struct {
	about  string
	data   string
	charms map[string]charm.Charm

	errors []string
}{{
	about:  "no charms",
	data:   mediawikiBundle,
	charms: map[string]charm.Charm{},
	errors: []string{
		`application "mediawiki" refers to non-existent charm "cs:precise/mediawiki-10"`,
		`application "mysql" refers to non-existent charm "cs:precise/mysql-28"`,
	},
}, {
	about: "all present and correct",
	data: `
applications:
    application1:
        charm: "test"
    application2:
        charm: "test"
    application3:
        charm: "test"
relations:
    - ["application1:prova", "application2:reqa"]
    - ["application1:reqa", "application3:prova"]
    - ["application3:provb", "application2:reqb"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
}, {
	about: "undefined relations",
	data: `
applications:
    application1:
        charm: "test"
    application2:
        charm: "test"
relations:
    - ["application1:prova", "application2:blah"]
    - ["application1:blah", "application2:prova"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`charm "test" used by application "application1" does not define relation "blah"`,
		`charm "test" used by application "application2" does not define relation "blah"`,
	},
}, {
	about: "undefined applications",
	data: `
applications:
    application1:
        charm: "test"
    application2:
        charm: "test"
relations:
    - ["unknown:prova", "application2:blah"]
    - ["application1:blah", "unknown:prova"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`relation ["application1:blah" "unknown:prova"] refers to application "unknown" not defined in this bundle`,
		`relation ["unknown:prova" "application2:blah"] refers to application "unknown" not defined in this bundle`,
	},
}, {
	about: "equal applications",
	data: `
applications:
    application1:
        charm: "test"
    application2:
        charm: "test"
relations:
    - ["application2:prova", "application2:reqa"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`relation ["application2:prova" "application2:reqa"] relates an application to itself`,
	},
}, {
	about: "provider to provider relation",
	data: `
applications:
    application1:
        charm: "test"
    application2:
        charm: "test"
relations:
    - ["application1:prova", "application2:prova"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`relation "application1:prova" to "application2:prova" relates provider to provider`,
	},
}, {
	about: "provider to provider relation",
	data: `
applications:
    application1:
        charm: "test"
    application2:
        charm: "test"
relations:
    - ["application1:reqa", "application2:reqa"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`relation "application1:reqa" to "application2:reqa" relates requirer to requirer`,
	},
}, {
	about: "interface mismatch",
	data: `
applications:
    application1:
        charm: "test"
    application2:
        charm: "test"
relations:
    - ["application1:reqa", "application2:provb"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`mismatched interface between "application2:provb" and "application1:reqa" ("b" vs "a")`,
	},
}, {
	about: "different charms",
	data: `
applications:
    application1:
        charm: "test1"
    application2:
        charm: "test2"
relations:
    - ["application1:reqa", "application2:prova"]
`,
	charms: map[string]charm.Charm{
		"test1": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
		"test2": testCharm("test", ""),
	},
	errors: []string{
		`charm "test2" used by application "application2" does not define relation "prova"`,
	},
}, {
	about: "ambiguous relation",
	data: `
applications:
    application1:
        charm: "test1"
    application2:
        charm: "test2"
relations:
    - [application1, application2]
`,
	charms: map[string]charm.Charm{
		"test1": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
		"test2": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`cannot infer endpoint between application1 and application2: ambiguous relation: application1 application2 could refer to "application1:prova application2:reqa"; "application1:provb application2:reqb"; "application1:reqa application2:prova"; "application1:reqb application2:provb"`,
	},
}, {
	about: "relation using juju-info",
	data: `
applications:
    application1:
        charm: "provider"
    application2:
        charm: "requirer"
relations:
    - [application1, application2]
`,
	charms: map[string]charm.Charm{
		"provider": testCharm("provider", ""),
		"requirer": testCharm("requirer", "| req:juju-info"),
	},
}, {
	about: "ambiguous when implicit relations taken into account",
	data: `
applications:
    application1:
        charm: "provider"
    application2:
        charm: "requirer"
relations:
    - [application1, application2]
`,
	charms: map[string]charm.Charm{
		"provider": testCharm("provider", "provdb:db | "),
		"requirer": testCharm("requirer", "| reqdb:db reqinfo:juju-info"),
	},
}, {
	about: "half of relation left open",
	data: `
applications:
    application1:
        charm: "provider"
    application2:
        charm: "requirer"
relations:
    - ["application1:prova2", application2]
`,
	charms: map[string]charm.Charm{
		"provider": testCharm("provider", "prova1:a prova2:a | "),
		"requirer": testCharm("requirer", "| reqa:a"),
	},
}, {
	about: "duplicate relation between open and fully-specified relations",
	data: `
applications:
    application1:
        charm: "provider"
    application2:
        charm: "requirer"
relations:
    - ["application1:prova", "application2:reqa"]
    - ["application1", "application2"]
`,
	charms: map[string]charm.Charm{
		"provider": testCharm("provider", "prova:a | "),
		"requirer": testCharm("requirer", "| reqa:a"),
	},
	errors: []string{
		`relation ["application1" "application2"] is defined more than once`,
	},
}, {
	about: "configuration options specified",
	data: `
applications:
    application1:
        charm: "test"
        options:
            title: "some title"
            skill-level: 245
    application2:
        charm: "test"
        options:
            title: "another title"
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
}, {
	about: "invalid type for option",
	data: `
applications:
    application1:
        charm: "test"
        options:
            title: "some title"
            skill-level: "too much"
    application2:
        charm: "test"
        options:
            title: "another title"
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`cannot validate application "application1": option "skill-level" expected int, got "too much"`,
	},
}, {
	about: "unknown option",
	data: `
applications:
    application1:
        charm: "test"
        options:
            title: "some title"
            unknown-option: 2345
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`cannot validate application "application1": configuration option "unknown-option" not found in charm "test"`,
	},
}, {
	about: "multiple config problems",
	data: `
applications:
    application1:
        charm: "test"
        options:
            title: "some title"
            unknown-option: 2345
    application2:
        charm: "test"
        options:
            title: 123
            another-unknown: 2345
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`cannot validate application "application1": configuration option "unknown-option" not found in charm "test"`,
		`cannot validate application "application2": configuration option "another-unknown" not found in charm "test"`,
		`cannot validate application "application2": option "title" expected string, got 123`,
	},
}, {
	about: "subordinate charm with more than zero units",
	data: `
applications:
    testsub:
        charm: "testsub"
        num_units: 1
`,
	charms: map[string]charm.Charm{
		"testsub": testCharm("test-sub", ""),
	},
	errors: []string{
		`application "testsub" is subordinate but has non-zero num_units`,
	},
}, {
	about: "subordinate charm with more than one unit",
	data: `
applications:
    testsub:
        charm: "testsub"
        num_units: 1
`,
	charms: map[string]charm.Charm{
		"testsub": testCharm("test-sub", ""),
	},
	errors: []string{
		`application "testsub" is subordinate but has non-zero num_units`,
	},
}, {
	about: "subordinate charm with to-clause",
	data: `
applications:
    testsub:
        charm: "testsub"
        to: [0]
machines:
    0:
`,
	charms: map[string]charm.Charm{
		"testsub": testCharm("test-sub", ""),
	},
	errors: []string{
		`application "testsub" is subordinate but specifies unit placement`,
		`too many units specified in unit placement for application "testsub"`,
	},
}, {
	about: "charm with unspecified units and more than one to: entry",
	data: `
applications:
    test:
        charm: "test"
        to: [0, 1]
machines:
    0:
    1:
`,
	errors: []string{
		`too many units specified in unit placement for application "test"`,
	},
}}

func (*bundleDataSuite) TestVerifyWithCharmsErrors(c *gc.C) {
	for i, test := range verifyWithCharmsErrorsTests {
		c.Logf("test %d: %s", i, test.about)
		assertVerifyErrors(c, test.data, test.charms, test.errors)
	}
}

var parsePlacementTests = []struct {
	placement string
	expect    *charm.UnitPlacement
	expectErr string
}{{
	placement: "lxc:application/0",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Application:   "application",
		Unit:          0,
	},
}, {
	placement: "lxc:application",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Application:   "application",
		Unit:          -1,
	},
}, {
	placement: "lxc:99",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Machine:       "99",
		Unit:          -1,
	},
}, {
	placement: "lxc:new",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Machine:       "new",
		Unit:          -1,
	},
}, {
	placement: "application/0",
	expect: &charm.UnitPlacement{
		Application: "application",
		Unit:        0,
	},
}, {
	placement: "application",
	expect: &charm.UnitPlacement{
		Application: "application",
		Unit:        -1,
	},
}, {
	placement: "application45",
	expect: &charm.UnitPlacement{
		Application: "application45",
		Unit:        -1,
	},
}, {
	placement: "99",
	expect: &charm.UnitPlacement{
		Machine: "99",
		Unit:    -1,
	},
}, {
	placement: "new",
	expect: &charm.UnitPlacement{
		Machine: "new",
		Unit:    -1,
	},
}, {
	placement: ":0",
	expectErr: `invalid placement syntax ":0"`,
}, {
	placement: "05",
	expectErr: `invalid placement syntax "05"`,
}, {
	placement: "new/2",
	expectErr: `invalid placement syntax "new/2"`,
}}

func (*bundleDataSuite) TestParsePlacement(c *gc.C) {
	for i, test := range parsePlacementTests {
		c.Logf("test %d: %q", i, test.placement)
		up, err := charm.ParsePlacement(test.placement)
		if test.expectErr != "" {
			c.Assert(err, gc.ErrorMatches, test.expectErr)
		} else {
			c.Assert(err, gc.IsNil)
			c.Assert(up, jc.DeepEquals, test.expect)
		}
	}
}

func (*bundleDataSuite) TestApplicationPlans(c *gc.C) {
	data := `
applications:
    application1:
        charm: "test"
        plan: "testisv/test"
    application2:
        charm: "test"
        plan: "testisv/test2"
    application3:
        charm: "test"
        plan: "default"
relations:
    - ["application1:prova", "application2:reqa"]
    - ["application1:reqa", "application3:prova"]
    - ["application3:provb", "application2:reqb"]
`

	bd, err := charm.ReadBundleData(strings.NewReader(data))
	c.Assert(err, gc.IsNil)

	c.Assert(bd.Applications, jc.DeepEquals, map[string]*charm.ApplicationSpec{
		"application1": &charm.ApplicationSpec{
			Charm: "test",
			Plan:  "testisv/test",
		},
		"application2": &charm.ApplicationSpec{
			Charm: "test",
			Plan:  "testisv/test2",
		},
		"application3": &charm.ApplicationSpec{
			Charm: "test",
			Plan:  "default",
		},
	})

}
