// Copyright 2014 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable"
)

type bundleDataSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&bundleDataSuite{})

const mediawikiBundle = `
series: precise
services:
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
    mysql:
        charm: "cs:precise/mysql-28"
        num_units: 2
        to: [0, mediawiki/0]
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
        constraints: "mem=8g"
        bindings:
            db: db
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
	about       string
	data        string
	expectedBD  *charm.BundleData
	expectedErr string
}{{
	about: "mediawiki",
	data:  mediawikiBundle,
	expectedBD: &charm.BundleData{
		Series: "precise",
		Services: map[string]*charm.ServiceSpec{
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
			},
			"mysql": {
				Charm:    "cs:precise/mysql-28",
				NumUnits: 2,
				To:       []string{"0", "mediawiki/0"},
				Options: map[string]interface{}{
					"binlog-format": "MIXED",
					"block-size":    5,
					"dataset-size":  "80%",
					"flavor":        "distro",
					"ha-bindiface":  "eth0",
					"ha-mcastport":  5411,
				},
				Annotations: map[string]string{
					"gui-x": "610",
					"gui-y": "255",
				},
				Constraints: "mem=8g",
				EndpointBindings: map[string]string{
					"db": "db",
				},
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
		c.Assert(bd, jc.DeepEquals, test.expectedBD)
	}
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
services:
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
		`invalid storage name "no_underscores" in service "ceph"`,
		`invalid storage "invalid-storage" in service "ceph-osd": bad storage constraint`,
		`machine "3" is not referred to by a placement directive`,
		`machine "bogus" is not referred to by a placement directive`,
		`invalid machine id "bogus" found in machines`,
		`invalid constraints "bad constraints" in machine "0": bad constraint`,
		`invalid charm URL in service "mediawiki": charm or bundle URL has invalid schema: "bogus:precise/mediawiki-10"`,
		`invalid constraints "bad constraints" in service "mysql": bad constraint`,
		`negative number of units specified on service "mediawiki"`,
		`too many units specified in unit placement for service "mysql"`,
		`placement "nowhere/3" refers to a service not defined in this bundle`,
		`placement "mediawiki/0" specifies a unit greater than the -4 unit(s) started by the target service`,
		`placement "2" refers to a machine not defined in this bundle`,
		`relation ["arble:bar"] has 1 endpoint(s), not 2`,
		`relation ["arble:bar" "mediawiki:db"] refers to service "arble" not defined in this bundle`,
		`relation ["mysql:foo" "mysql:bar"] relates a service to itself`,
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
		assertVerifyWithCharmsErrors(c, test.data, nil, test.errors)
	}
}

func assertVerifyWithCharmsErrors(c *gc.C, bundleData string, charms map[string]charm.Charm, expectErrors []string) {
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

	err = bd.VerifyWithCharms(validateConstraints, validateStorage, charms)
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
		bd.Services["mediawiki"].Charm = u
		err := bd.Verify(nil, nil)
		c.Assert(err, gc.IsNil, gc.Commentf("charm url %q", u))
	}
}

func (*bundleDataSuite) TestVerifyBundleUsingJujuInfoRelation(c *gc.C) {
	b := readBundleDir(c, "wordpress-with-logging")
	bd := b.Data()

	charms := map[string]charm.Charm{
		"wordpress": readCharmDir(c, "wordpress"),
		"mysql":     readCharmDir(c, "mysql"),
		"logging":   readCharmDir(c, "logging"),
	}
	err := bd.VerifyWithCharms(nil, nil, charms)
	c.Assert(err, gc.IsNil)
}

func (*bundleDataSuite) TestVerifyBundleUsingJujuInfoRelationBindingFail(c *gc.C) {
	b := readBundleDir(c, "wordpress-with-logging")
	bd := b.Data()

	charms := map[string]charm.Charm{
		"wordpress": readCharmDir(c, "wordpress"),
		"mysql":     readCharmDir(c, "mysql"),
		"logging":   readCharmDir(c, "logging"),
	}
	bd.Services["wordpress"].EndpointBindings["foo"] = "bar"
	err := bd.VerifyWithCharms(nil, nil, charms)

	c.Assert(err, gc.ErrorMatches,
		"service \"wordpress\" wants to bind endpoint \"foo\" to space \"bar\", "+
			"but the endpoint is not defined by the charm")
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
		`service "mediawiki" refers to non-existent charm "cs:precise/mediawiki-10"`,
		`service "mysql" refers to non-existent charm "cs:precise/mysql-28"`,
	},
}, {
	about: "all present and correct",
	data: `
services:
    service1:
        charm: "test"
    service2:
        charm: "test"
    service3:
        charm: "test"
relations:
    - ["service1:prova", "service2:reqa"]
    - ["service1:reqa", "service3:prova"]
    - ["service3:provb", "service2:reqb"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
}, {
	about: "undefined relations",
	data: `
services:
    service1:
        charm: "test"
    service2:
        charm: "test"
relations:
    - ["service1:prova", "service2:blah"]
    - ["service1:blah", "service2:prova"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`charm "test" used by service "service1" does not define relation "blah"`,
		`charm "test" used by service "service2" does not define relation "blah"`,
	},
}, {
	about: "undefined services",
	data: `
services:
    service1:
        charm: "test"
    service2:
        charm: "test"
relations:
    - ["unknown:prova", "service2:blah"]
    - ["service1:blah", "unknown:prova"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`relation ["service1:blah" "unknown:prova"] refers to service "unknown" not defined in this bundle`,
		`relation ["unknown:prova" "service2:blah"] refers to service "unknown" not defined in this bundle`,
	},
}, {
	about: "equal services",
	data: `
services:
    service1:
        charm: "test"
    service2:
        charm: "test"
relations:
    - ["service2:prova", "service2:reqa"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`relation ["service2:prova" "service2:reqa"] relates a service to itself`,
	},
}, {
	about: "provider to provider relation",
	data: `
services:
    service1:
        charm: "test"
    service2:
        charm: "test"
relations:
    - ["service1:prova", "service2:prova"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`relation "service1:prova" to "service2:prova" relates provider to provider`,
	},
}, {
	about: "provider to provider relation",
	data: `
services:
    service1:
        charm: "test"
    service2:
        charm: "test"
relations:
    - ["service1:reqa", "service2:reqa"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`relation "service1:reqa" to "service2:reqa" relates requirer to requirer`,
	},
}, {
	about: "interface mismatch",
	data: `
services:
    service1:
        charm: "test"
    service2:
        charm: "test"
relations:
    - ["service1:reqa", "service2:provb"]
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`mismatched interface between "service2:provb" and "service1:reqa" ("b" vs "a")`,
	},
}, {
	about: "different charms",
	data: `
services:
    service1:
        charm: "test1"
    service2:
        charm: "test2"
relations:
    - ["service1:reqa", "service2:prova"]
`,
	charms: map[string]charm.Charm{
		"test1": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
		"test2": testCharm("test", ""),
	},
	errors: []string{
		`charm "test2" used by service "service2" does not define relation "prova"`,
	},
}, {
	about: "ambiguous relation",
	data: `
services:
    service1:
        charm: "test1"
    service2:
        charm: "test2"
relations:
    - [service1, service2]
`,
	charms: map[string]charm.Charm{
		"test1": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
		"test2": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`cannot infer endpoint between service1 and service2: ambiguous relation: service1 service2 could refer to "service1:prova service2:reqa"; "service1:provb service2:reqb"; "service1:reqa service2:prova"; "service1:reqb service2:provb"`,
	},
}, {
	about: "relation using juju-info",
	data: `
services:
    service1:
        charm: "provider"
    service2:
        charm: "requirer"
relations:
    - [service1, service2]
`,
	charms: map[string]charm.Charm{
		"provider": testCharm("provider", ""),
		"requirer": testCharm("requirer", "| req:juju-info"),
	},
}, {
	about: "ambiguous when implicit relations taken into account",
	data: `
services:
    service1:
        charm: "provider"
    service2:
        charm: "requirer"
relations:
    - [service1, service2]
`,
	charms: map[string]charm.Charm{
		"provider": testCharm("provider", "provdb:db | "),
		"requirer": testCharm("requirer", "| reqdb:db reqinfo:juju-info"),
	},
}, {
	about: "half of relation left open",
	data: `
services:
    service1:
        charm: "provider"
    service2:
        charm: "requirer"
relations:
    - ["service1:prova2", service2]
`,
	charms: map[string]charm.Charm{
		"provider": testCharm("provider", "prova1:a prova2:a | "),
		"requirer": testCharm("requirer", "| reqa:a"),
	},
}, {
	about: "duplicate relation between open and fully-specified relations",
	data: `
services:
    service1:
        charm: "provider"
    service2:
        charm: "requirer"
relations:
    - ["service1:prova", "service2:reqa"]
    - ["service1", "service2"]
`,
	charms: map[string]charm.Charm{
		"provider": testCharm("provider", "prova:a | "),
		"requirer": testCharm("requirer", "| reqa:a"),
	},
	errors: []string{
		`relation ["service1" "service2"] is defined more than once`,
	},
}, {
	about: "configuration options specified",
	data: `
services:
    service1:
        charm: "test"
        options:
            title: "some title"
            skill-level: 245
    service2:
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
services:
    service1:
        charm: "test"
        options:
            title: "some title"
            skill-level: "too much"
    service2:
        charm: "test"
        options:
            title: "another title"
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`cannot validate service "service1": option "skill-level" expected int, got "too much"`,
	},
}, {
	about: "unknown option",
	data: `
services:
    service1:
        charm: "test"
        options:
            title: "some title"
            unknown-option: 2345
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`cannot validate service "service1": configuration option "unknown-option" not found in charm "test"`,
	},
}, {
	about: "multiple config problems",
	data: `
services:
    service1:
        charm: "test"
        options:
            title: "some title"
            unknown-option: 2345
    service2:
        charm: "test"
        options:
            title: 123
            another-unknown: 2345
`,
	charms: map[string]charm.Charm{
		"test": testCharm("test", "prova:a provb:b | reqa:a reqb:b"),
	},
	errors: []string{
		`cannot validate service "service1": configuration option "unknown-option" not found in charm "test"`,
		`cannot validate service "service2": configuration option "another-unknown" not found in charm "test"`,
		`cannot validate service "service2": option "title" expected string, got 123`,
	},
}, {
	about: "subordinate charm with more than zero units",
	data: `
services:
    testsub:
        charm: "testsub"
        num_units: 1
`,
	charms: map[string]charm.Charm{
		"testsub": testCharm("test-sub", ""),
	},
	errors: []string{
		`service "testsub" is subordinate but has non-zero num_units`,
	},
}, {
	about: "subordinate charm with more than one unit",
	data: `
services:
    testsub:
        charm: "testsub"
        num_units: 1
`,
	charms: map[string]charm.Charm{
		"testsub": testCharm("test-sub", ""),
	},
	errors: []string{
		`service "testsub" is subordinate but has non-zero num_units`,
	},
}, {
	about: "subordinate charm with to-clause",
	data: `
services:
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
		`service "testsub" is subordinate but specifies unit placement`,
		`too many units specified in unit placement for service "testsub"`,
	},
}, {
	about: "charm with unspecified units and more than one to: entry",
	data: `
services:
    test:
        charm: "test"
        to: [0, 1]
machines:
    0:
    1:
`,
	errors: []string{
		`too many units specified in unit placement for service "test"`,
	},
}}

func (*bundleDataSuite) TestVerifyWithCharmsErrors(c *gc.C) {
	for i, test := range verifyWithCharmsErrorsTests {
		c.Logf("test %d: %s", i, test.about)
		assertVerifyWithCharmsErrors(c, test.data, test.charms, test.errors)
	}
}

var parsePlacementTests = []struct {
	placement string
	expect    *charm.UnitPlacement
	expectErr string
}{{
	placement: "lxc:service/0",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Service:       "service",
		Unit:          0,
	},
}, {
	placement: "lxc:service",
	expect: &charm.UnitPlacement{
		ContainerType: "lxc",
		Service:       "service",
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
	placement: "service/0",
	expect: &charm.UnitPlacement{
		Service: "service",
		Unit:    0,
	},
}, {
	placement: "service",
	expect: &charm.UnitPlacement{
		Service: "service",
		Unit:    -1,
	},
}, {
	placement: "service45",
	expect: &charm.UnitPlacement{
		Service: "service45",
		Unit:    -1,
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
