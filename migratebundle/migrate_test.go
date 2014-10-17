package migratebundle

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/errgo.v1"
	"gopkg.in/yaml.v1"

	"gopkg.in/juju/charm.v4"
)

var _ = gc.Suite(&migrateSuite{})

type migrateSuite struct{}

// The charm data cache caches results from
// fetching charms from the charm store.
// If the update-charms flag is specified, the
// contents of charmDataCache is written to
// allcharms.json.gz; otherwise the contents
// of allcharms.json are read and the charm
// store is not touched.
//
var (
	charmDataCacheMutex sync.Mutex
	charmDataCache      = make(map[string]*charmData)
)

var updateCharms = flag.Bool("update-charms", false, "fetch and update local charms for test bundles")

const charmCacheFile = "allcharms.json.gz"

func (*migrateSuite) SetUpSuite(c *gc.C) {
	if *updateCharms {
		charm.CacheDir = c.MkDir()
		return
	}
	f, err := os.Open(charmCacheFile)
	if err != nil {
		c.Logf("cannot open charms data: %v", err)
		return
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	c.Assert(err, gc.IsNil)
	dec := json.NewDecoder(gzr)
	err = dec.Decode(&charmDataCache)
	c.Assert(err, gc.IsNil)
}

func (*migrateSuite) TearDownSuite(c *gc.C) {
	if !*updateCharms {
		return
	}
	data, err := json.Marshal(charmDataCache)
	c.Assert(err, gc.IsNil)
	f, err := os.Create(charmCacheFile)
	c.Assert(err, gc.IsNil)
	defer f.Close()
	gzw := gzip.NewWriter(f)
	defer gzw.Close()
	_, err = gzw.Write(data)
	c.Assert(err, gc.IsNil)
}

var migrateTests = []struct {
	about       string
	bundles     string
	expect      map[string]*charm.BundleData
	expectError string
}{{
	about: "single bundle, no relations cs:~jorge/bundle/wordpress",
	bundles: `
		|wordpress-simple: 
		|    series: precise
		|    services: 
		|        wordpress: 
		|            charm: "cs:precise/wordpress-20"
		|            num_units: 1
		|            options: 
		|                debug: "no"
		|                engine: nginx
		|                tuning: single
		|                "wp-content": ""
		|            annotations: 
		|                "gui-x": 529
		|                "gui-y": -97
		|        mysql: 
		|            charm: "cs:precise/mysql-28"
		|            num_units: 2
		|            options: 
		|                "binlog-format": MIXED
		|                "block-size": 5
		|                "dataset-size": "80%"
		|                flavor: distro
		|                "query-cache-size": -1
		|                "query-cache-type": "OFF"
		|                vip_iface: eth0
		|            annotations: 
		|                "gui-x": 530
		|                "gui-y": 185
		|`,
	expect: map[string]*charm.BundleData{
		"wordpress-simple": {
			Series: "precise",
			Services: map[string]*charm.ServiceSpec{
				"wordpress": {
					Charm:    "cs:precise/wordpress-20",
					NumUnits: 1,
					Options: map[string]interface{}{
						"debug":      "no",
						"engine":     "nginx",
						"tuning":     "single",
						"wp-content": "",
					},
					Annotations: map[string]string{
						"gui-x": "529",
						"gui-y": "-97",
					},
				},
				"mysql": {
					Charm:    "cs:precise/mysql-28",
					NumUnits: 2,
					Options: map[string]interface{}{
						"binlog-format":    "MIXED",
						"block-size":       5,
						"dataset-size":     "80%",
						"flavor":           "distro",
						"query-cache-size": -1,
						"query-cache-type": "OFF",
						"vip_iface":        "eth0",
					},
					Annotations: map[string]string{
						"gui-x": "530",
						"gui-y": "185",
					},
				},
			},
		},
	},
}, {
	about: "missing num_units interpreted as single unit",
	bundles: `
		|wordpress-simple: 
		|    services: 
		|        wordpress: 
		|            charm: "cs:precise/wordpress-20"
		|`,
	expect: map[string]*charm.BundleData{
		"wordpress-simple": {
			Services: map[string]*charm.ServiceSpec{
				"wordpress": {
					Charm:    "cs:precise/wordpress-20",
					NumUnits: 1,
				},
			},
		},
	},
}, {
	about: "missing charm taken from service name",
	bundles: `
		|wordpress-simple: 
		|    services: 
		|        wordpress: 
		|`,
	expect: map[string]*charm.BundleData{
		"wordpress-simple": {
			Services: map[string]*charm.ServiceSpec{
				"wordpress": {
					Charm:    "wordpress",
					NumUnits: 1,
				},
			},
		},
	},
}, {
	about: "services with placement directives",
	bundles: `
		|wordpress: 
		|    services: 
		|        wordpress1:
		|            num_units: 1
		|            to: 0
		|        wordpress2:
		|            num_units: 1
		|            to: kvm:0
		|        wordpress3:
		|            num_units: 1
		|            to: mysql
		|        wordpress4:
		|            num_units: 1
		|            to: kvm:mysql
		|        mysql:
		|	    num_units: 1
		|`,
	expect: map[string]*charm.BundleData{
		"wordpress": {
			Services: map[string]*charm.ServiceSpec{
				"wordpress1": {
					Charm:    "wordpress1",
					NumUnits: 1,
					To:       []string{"0"},
				},
				"wordpress2": {
					Charm:    "wordpress2",
					NumUnits: 1,
					To:       []string{"kvm:0"},
				},
				"wordpress3": {
					Charm:    "wordpress3",
					NumUnits: 1,
					To:       []string{"mysql"},
				},
				"wordpress4": {
					Charm:    "wordpress4",
					NumUnits: 1,
					To:       []string{"kvm:mysql"},
				},
				"mysql": {
					Charm:    "mysql",
					NumUnits: 1,
				},
			},
			Machines: map[string]*charm.MachineSpec{
				"0": {},
			},
		},
	},
}, {
	about: "service with single indirect placement directive",
	bundles: `
		|wordpress: 
		|    services: 
		|        wordpress:
		|            to: kvm:0
		|`,
	expect: map[string]*charm.BundleData{
		"wordpress": {
			Services: map[string]*charm.ServiceSpec{
				"wordpress": {
					Charm:    "wordpress",
					NumUnits: 1,
					To:       []string{"kvm:0"},
				},
			},
			Machines: map[string]*charm.MachineSpec{
				"0": {},
			},
		},
	},
}, {
	about: "service with invalid placement directive",
	bundles: `
		|wordpress: 
		|    services: 
		|        wordpress:
		|            to: kvm::0
		|`,
	expectError: `bundle migration failed for "wordpress": cannot parse 'to' placment clause "kvm::0": invalid placement syntax "kvm::0"`,
}, {
	about: "service with inheritance",
	bundles: `
		|wordpress:
		|    inherits: base
		|    services: 
		|        wordpress:
		|            charm: precise/wordpress
		|            annotations:
		|                 foo: yes
		|                 base: arble
		|base:
		|    services:
		|        logging:
		|             charm: precise/logging
		|        wordpress:
		|            annotations:
		|                 foo: bar
		|                 base: arble
		|`,
	expect: map[string]*charm.BundleData{
		"wordpress": {
			Services: map[string]*charm.ServiceSpec{
				"wordpress": {
					Charm:    "precise/wordpress",
					NumUnits: 1,
					Annotations: map[string]string{
						"foo":  "yes",
						"base": "arble",
					},
				},
				"logging": {
					Charm:    "precise/logging",
					NumUnits: 1,
				},
			},
		},
		"base": {
			Services: map[string]*charm.ServiceSpec{
				"logging": {
					Charm:    "precise/logging",
					NumUnits: 1,
				},
				"wordpress": {
					Charm:    "wordpress",
					NumUnits: 1,
					Annotations: map[string]string{
						"foo":  "bar",
						"base": "arble",
					},
				},
			},
		},
	},
}, {
	about: "open relations",
	bundles: `
		|wordpress:
		|    services:
		|        wordpress:
		|            charm: precise/wordpress
		|        mysql:
		|            charm: precise/mysql
		|        logging:
		|            charm: precise/logging
		|        monitoring:
		|            charm: precise/monitor
		|    relations:
		|        - [wordpress, mysql]
		|        - [logging, [mysql, wordpress]]
		|        - [monitoring, wordpress]
		|`,
	expect: map[string]*charm.BundleData{
		"wordpress": {
			Services: map[string]*charm.ServiceSpec{
				"wordpress": {
					Charm:    "precise/wordpress",
					NumUnits: 1,
				},
				"mysql": {
					Charm:    "precise/mysql",
					NumUnits: 1,
				},
				"logging": {
					Charm:    "precise/logging",
					NumUnits: 1,
				},
				"monitoring": {
					Charm:    "precise/monitor",
					NumUnits: 1,
				},
			},
			Relations: [][]string{
				{"wordpress", "mysql"},
				{"logging", "mysql"},
				{"logging", "wordpress"},
				{"monitoring", "wordpress"},
			},
		},
	},
}}

func (*migrateSuite) TestMigrate(c *gc.C) {
	for i, test := range migrateTests {
		c.Logf("test %d: %s", i, test.about)
		result, err := Migrate(unbeautify(test.bundles), nil)
		if test.expectError != "" {
			c.Assert(err, gc.ErrorMatches, test.expectError)
		} else {
			c.Assert(err, gc.IsNil)
			c.Assert(result, jc.DeepEquals, test.expect)
		}
	}
}

func (*migrateSuite) TestMigrateAll(c *gc.C) {
	c.ExpectFailure("all bundles do not migrate successfully")
	passed, total := 0, 0
	doAllBundles(c, func(c *gc.C, id string, data []byte) {
		c.Logf("\nmigrate test %s", id)
		ok := true
		bundles, err := Migrate(data, nil)
		if err != nil {
			c.Logf("cannot migrate: %v", err)
			ok = false
		}
		for _, bundle := range bundles {
			ok = checkBundleData(c, bundle) && ok
		}
		if ok {
			passed++
		}
		total++
	})
	c.Logf("%d/%d passed", passed, total)
	c.Check(passed, gc.Equals, total)
}

func checkBundleData(c *gc.C, bd *charm.BundleData) bool {
	charms := make(map[string]charm.Charm)
	ok := true
	for _, svc := range bd.Services {
		id, err := charm.ParseReference(svc.Charm)
		if err != nil {
			ok = false
			c.Logf("cannot parse %q: %v", svc.Charm, err)
			continue
		}
		if id.Series == "" {
			id.Series = bd.Series
		}
		ch, err := getCharm(id)
		if err != nil {
			ok = false
			c.Logf("cannot find %q: %v", id, err)
			continue
		}
		charms[svc.Charm] = ch
	}
	if ok {
		if err := bd.VerifyWithCharms(nil, charms); err != nil {
			for _, err := range err.(*charm.VerificationError).Errors {
				c.Logf("verification error: %v", err)
			}
			ok = false
		}
	}
	return ok
}

var inheritTests = []struct {
	about       string
	bundle      string
	base        string
	baseName    string
	expect      string
	expectError string
}{{
	about:       "inherited-from not found",
	bundle:      `inherits: non-existent`,
	expectError: `inherited-from bundle "non-existent" not found`,
}, {
	about:       "bad inheritance #1",
	bundle:      `inherits: 200`,
	expectError: `bad inherits clause 200`,
}, {
	about:       "bad inheritance #2",
	bundle:      `inherits: [10]`,
	expectError: `bad inherits clause .*`,
}, {
	about:       "bad inheritance #3",
	bundle:      `inherits: ['a', 'b']`,
	expectError: `bad inherits clause .*`,
}, {
	about: "inherit everything",
	bundle: `
		|inherits: base
	`,
	baseName: "base",
	base: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
	`,
	expect: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
	`,
}, {
	about: "inherit everything, specified as list",
	bundle: `
		|inherits: [base]
	`,
	baseName: "base",
	base: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
	`,
	expect: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
	`,
}, {
	about: "different base name",
	bundle: `
		|inherits: something
	`,
	baseName: "something",
	base: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
	`,
	expect: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
	`,
}, {
	about: "override series",
	bundle: `
		|inherits: base
		|series: trusty
	`,
	baseName: "base",
	base: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
	`,
	expect: `
		|series: trusty
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
	`,
}, {
	about: "override wordpress charm",
	bundle: `
		|inherits: base
		|services:
		|    wordpress:
		|        charm: 'cs:quantal/different'
	`,
	baseName: "base",
	base: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: "cs:precise/wordpress"
		|        options:
		|            foo: bar
	`,
	expect: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: "cs:quantal/different"
		|        options:
		|            foo: bar
	`,
}, {
	about: "override to clause",
	bundle: `
		|inherits: base
		|services:
		|    wordpress:
		|        to: 0
	`,
	baseName: "base",
	base: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
		|        options:
		|            foo: bar
	`,
	expect: `
		|series: precise
		|services:
		|    wordpress:
		|        charm: 'cs:precise/wordpress'
		|        options:
		|            foo: bar
		|        to: 0
	`,
}, {
	about: "deep inheritance",
	bundle: `
		|inherits: base
	`,
	baseName: "base",
	base: `
		|inherits: "other"
	`,
	expectError: `only a single level of inheritance is supported`,
}}

var otherBundle = parseBundle(`
	|series: quantal
	|overrides:
	|  something: other
`)

func (*migrateSuite) TestInherit(c *gc.C) {
	for i, test := range inheritTests {
		c.Logf("test %d: %s", i, test.about)
		bundle := parseBundle(test.bundle)
		base := parseBundle(test.base)
		expect := parseBundle(test.expect)
		// Add another bundle so we know that is
		bundles := map[string]*legacyBundle{
			test.baseName: base,
			"other":       otherBundle,
		}
		b, err := inherit(bundle, bundles)
		if test.expectError != "" {
			c.Check(err, gc.ErrorMatches, test.expectError)
		} else {
			c.Assert(err, gc.IsNil)
			c.Assert(b, jc.DeepEquals, expect)
		}
	}
}

func (s *migrateSuite) TestNoNameClashes(c *gc.C) {
	nameCounts := make(map[string]int)
	doAllBundles(c, func(c *gc.C, id string, data []byte) {
		nameCounts[id]++
	})
	// There are actually two name clashes in the real
	// in-the-wild bundles:
	//     cs:~charmers/bundle/mediawiki-scalable
	//     cs:~charmers/bundle/mongodb-cluster
	// Both of these actually fit with our proposed scheme,
	// because they're (almost) identical with the bundles
	// within mediawiki and mongodb respectively.
	//
	// So we discount them from our example bundles.
	delete(nameCounts, "cs:~charmers/bundle/mongodb-cluster")
	delete(nameCounts, "cs:~charmers/bundle/mediawiki-scalable")

	doAllBundles(c, func(c *gc.C, id string, data []byte) {
		var bundles map[string]*legacyBundle
		err := yaml.Unmarshal(data, &bundles)
		c.Assert(err, gc.IsNil)
		if len(bundles) == 1 {
			return
		}
		for name := range bundles {
			subId := id + "-" + name
			nameCounts[subId]++
		}
	})
	for name, count := range nameCounts {
		if count != 1 {
			c.Errorf("%d clashes at %s", count-1, name)
		}
	}
}

func (s *migrateSuite) TestReversible(c *gc.C) {
	doAllBundles(c, s.testReversible)
}

func (*migrateSuite) testReversible(c *gc.C, id string, data []byte) {
	var bundles map[string]*legacyBundle
	err := yaml.Unmarshal(data, &bundles)
	c.Assert(err, gc.IsNil)
	for _, b := range bundles {
		if len(b.Relations) == 0 {
			b.Relations = nil
		}
	}
	var allInterface interface{}
	err = yaml.Unmarshal(data, &allInterface)
	c.Assert(err, gc.IsNil)
	all, ok := allInterface.(map[interface{}]interface{})
	c.Assert(ok, gc.Equals, true)
	for _, b := range all {
		b := ymap(b)
		// Remove empty relations line.
		if rels, ok := b["relations"].([]interface{}); ok && len(rels) == 0 {
			delete(b, "relations")
		}
		// Convert all annotation values and "to" values
		// to strings.
		// Strictly speaking this means that the bundles
		// are non-reversible, but juju converts annotations
		// to string anyway, so it doesn't matter.
		for _, svc := range ymap(b["services"]) {
			svc := ymap(svc)
			annot := ymap(svc["annotations"])
			for key, val := range annot {
				if _, ok := val.(string); !ok {
					annot[key] = fmt.Sprint(val)
				}
			}
			if to, ok := svc["to"]; ok {
				svc["to"] = fmt.Sprint(to)
			}
		}

	}
	data1, err := yaml.Marshal(bundles)
	c.Assert(err, gc.IsNil)
	var all1 interface{}
	err = yaml.Unmarshal(data1, &all1)
	c.Assert(err, gc.IsNil)
	c.Assert(all1, jc.DeepEquals, all)
}

// ymap returns the default form of a map
// when unmarshaled by YAML.
func ymap(v interface{}) map[interface{}]interface{} {
	if v == nil {
		return nil
	}
	return v.(map[interface{}]interface{})
}

// doAllBundles calls the given function for each bundle
// in all the available test bundles.
func doAllBundles(c *gc.C, f func(c *gc.C, id string, data []byte)) {
	a := openAllBundles()
	defer a.Close()
	for {
		title, data, err := a.readSection()
		if len(data) > 0 {
			f(c, title, data)
		}
		if err != nil {
			c.Assert(errgo.Cause(err), gc.Equals, io.EOF)
			break
		}
	}
}

type allBundles struct {
	file *os.File
	r    *bufio.Reader
}

func openAllBundles() *allBundles {
	f, err := os.Open("allbundles.txt.gz")
	if err != nil {
		log.Fatal(err)
	}
	gzr, err := gzip.NewReader(f)
	if err != nil {
		log.Fatal(err)
	}
	r := bufio.NewReader(gzr)
	return &allBundles{
		file: f,
		r:    r,
	}
}

func (a *allBundles) Close() error {
	return a.file.Close()
}

// sectionMarker delimits a section in the bundles file.
// Note that no bundles contain non-ASCII characters
// so the first byte of this string is a sufficient
// sentinel.
const sectionMarker = "¶ "

func (a *allBundles) readSection() (title string, data []byte, err error) {
	title, err = a.r.ReadString('\n')
	if err != nil {
		return "", nil, err
	}
	if !strings.HasPrefix(title, sectionMarker) || !strings.HasSuffix(title, "\n") {
		return "", nil, fmt.Errorf("invalid title line %q", title)
	}
	title = strings.TrimPrefix(title, sectionMarker)
	title = strings.TrimSuffix(title, "\n")
	for {
		c, err := a.r.ReadByte()
		switch {
		case err == io.EOF:
			return title, data, nil
		case err != nil:
			return "", nil, err
		case c == sectionMarker[0]:
			a.r.UnreadByte()
			return title, data, nil
		}
		data = append(data, c)
	}
}

func parseBundle(s string) *legacyBundle {
	var b *legacyBundle
	err := yaml.Unmarshal(unbeautify(s), &b)
	if err != nil {
		panic(fmt.Errorf("cannot unmarshal %q: %v", s, err))
	}
	return b
}

// indentReplacer deletes tabs and | beautifier characters.
var indentReplacer = strings.NewReplacer("\t", "", "|", "")

// unbeautify strip the tabs and | characters that
// we use to make the tests look nicer.
func unbeautify(s string) []byte {
	return []byte(indentReplacer.Replace(s))
}

func noCharms(id *charm.Reference) (*charm.Meta, error) {
	return nil, fmt.Errorf("charm %q not found", id)
}

func getCharm(id *charm.Reference) (charm.Charm, error) {
	url, err := id.URL("")
	if err != nil {
		return nil, fmt.Errorf("cannot make URL from %q: %v", id, err)
	}
	charmDataCacheMutex.Lock()
	defer charmDataCacheMutex.Unlock()
	if m, ok := charmDataCache[url.String()]; ok || !*updateCharms {
		if m == nil {
			return nil, fmt.Errorf("charm %q not found in cache", id)
		}
		return m, nil
	}
	log.Printf("getting %s", url)
	ch, err := charm.Store.Get(url)
	if err != nil {
		charmDataCache[url.String()] = nil
		return nil, err
	}
	chData := &charmData{
		Meta_:    ch.Meta(),
		Config_:  ch.Config(),
		Metrics_: ch.Metrics(),
	}
	charmDataCache[url.String()] = chData
	return chData, nil
}

type charmData struct {
	Meta_    *charm.Meta    `json:"Meta"`
	Config_  *charm.Config  `json:"Config"`
	Metrics_ *charm.Metrics `json:"Metrics"`
}

func (c *charmData) Meta() *charm.Meta {
	return c.Meta_
}

func (c *charmData) Metrics() *charm.Metrics {
	return c.Metrics_
}

func (c *charmData) Config() *charm.Config {
	return c.Config_
}

func (c *charmData) Actions() *charm.Actions {
	return nil
}

func (c *charmData) Revision() int {
	return 0
}
