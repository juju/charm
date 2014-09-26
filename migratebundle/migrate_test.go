package migratebundle

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/juju/errgo"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/yaml.v2"
)

var _ = gc.Suite(&migrateSuite{})

type migrateSuite struct{}

var inheritTests = []struct {
	about    string
	bundle   string
	base     string
	baseName string
	expect   string
	error    string
}{{
	about:  "inherited-from not found",
	bundle: `inherits: non-existent`,
	error:  `inherited-from bundle "non-existent" not found`,
}, {
	about:  "bad inheritance #1",
	bundle: `inherits: 200`,
	error:  `bad inherits clause 200`,
}, {
	about:  "bad inheritance #2",
	bundle: `inherits: [10]`,
	error:  `bad inherits clause .*`,
}, {
	about:  "bad inheritance #3",
	bundle: `inherits: ['a', 'b']`,
	error:  `bad inherits clause .*`,
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
}}

// indentReplacer deletes tabs and | beautifier characters.
var indentReplacer = strings.NewReplacer("\t", "", "|", "")

func parseBundle(c *gc.C, s string) *legacyBundle {
	s = indentReplacer.Replace(s)
	var b *legacyBundle
	err := yaml.Unmarshal([]byte(s), &b)
	c.Assert(err, gc.IsNil)
	return b
}

func (*migrateSuite) TestInherit(c *gc.C) {
	for i, test := range inheritTests {
		c.Logf("test %d: %s", i, test.about)
		bundle := parseBundle(c, test.bundle)
		base := parseBundle(c, test.base)
		expect := parseBundle(c, test.expect)
		bundles := map[string]*legacyBundle{
			test.baseName: base,
		}
		b, err := inherit(bundle, bundles)
		if test.error != "" {
			c.Check(err, gc.ErrorMatches, test.error)
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
		b, ok := b.(map[interface{}]interface{})
		if !ok {
			c.Fatalf("bundle without map; actually %T", b)
		}
		if rels, ok := b["relations"].([]interface{}); ok && len(rels) == 0 {
			delete(b, "relations")
		}
	}
	data1, err := yaml.Marshal(bundles)
	c.Assert(err, gc.IsNil)
	var all1 interface{}
	err = yaml.Unmarshal(data1, &all1)
	c.Assert(err, gc.IsNil)
	c.Assert(all1, jc.DeepEquals, all)
}

// doAllBundles calls the given function for each bundle
// in all the available test bundles.
func doAllBundles(c *gc.C, f func(c *gc.C, id string, data []byte)) {
	a := openAllBundles()
	defer a.Close()
	for {
		title, data, err := a.readSection()
		if len(data) > 0 {
			c.Logf("charm %s", title)
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
