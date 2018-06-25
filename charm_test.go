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
	stdtesting "testing"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/fs"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v6"
	"gopkg.in/yaml.v2"
)

func Test(t *stdtesting.T) {
	gc.TestingT(t)
}

type CharmSuite struct {
	testing.CleanupSuite
}

var _ = gc.Suite(&CharmSuite{})

func (s *CharmSuite) TestReadCharm(c *gc.C) {
	ch, err := charm.ReadCharm(charmDirPath(c, "dummy"))
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Meta().Name, gc.Equals, "dummy")

	bPath := archivePath(c, readCharmDir(c, "dummy"))
	ch, err = charm.ReadCharm(bPath)
	c.Assert(err, gc.IsNil)
	c.Assert(ch.Meta().Name, gc.Equals, "dummy")
}

func (s *CharmSuite) TestReadCharmDirError(c *gc.C) {
	ch, err := charm.ReadCharm(c.MkDir())
	c.Assert(err, gc.NotNil)
	c.Assert(ch, gc.Equals, nil)
}

func (s *CharmSuite) TestReadCharmArchiveError(c *gc.C) {
	path := filepath.Join(c.MkDir(), "path")
	err := ioutil.WriteFile(path, []byte("foo"), 0644)
	c.Assert(err, gc.IsNil)
	ch, err := charm.ReadCharm(path)
	c.Assert(err, gc.NotNil)
	c.Assert(ch, gc.Equals, nil)
}

func (s *CharmSuite) TestSeriesToUse(c *gc.C) {
	tests := []struct {
		series          string
		supportedSeries []string
		seriesToUse     string
		err             string
	}{{
		series: "",
		err:    "series not specified and charm does not define any",
	}, {
		series:      "trusty",
		seriesToUse: "trusty",
	}, {
		series:          "trusty",
		supportedSeries: []string{"precise", "trusty"},
		seriesToUse:     "trusty",
	}, {
		series:          "",
		supportedSeries: []string{"precise", "trusty"},
		seriesToUse:     "precise",
	}, {
		series:          "wily",
		supportedSeries: []string{"precise", "trusty"},
		err:             `series "wily" not supported by charm.*`,
	}}
	for _, test := range tests {
		series, err := charm.SeriesForCharm(test.series, test.supportedSeries)
		if test.err != "" {
			c.Assert(err, gc.ErrorMatches, test.err)
			continue
		}
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(series, jc.DeepEquals, test.seriesToUse)
	}
}

func (s *CharmSuite) IsUnsupportedSeriesError(c *gc.C) {
	err := charm.NewUnsupportedSeriesError("series", []string{"supported"})
	c.Assert(charm.IsUnsupportedSeriesError(err), jc.IsTrue)
	c.Assert(charm.IsUnsupportedSeriesError(fmt.Errorf("foo")), jc.IsFalse)
}

func (s *CharmSuite) IsMissingSeriesError(c *gc.C) {
	err := charm.MissingSeriesError()
	c.Assert(charm.IsMissingSeriesError(err), jc.IsTrue)
	c.Assert(charm.IsMissingSeriesError(fmt.Errorf("foo")), jc.IsFalse)
}

func checkDummy(c *gc.C, f charm.Charm, path string) {
	c.Assert(f.Revision(), gc.Equals, 1)
	c.Assert(f.Meta().Name, gc.Equals, "dummy")
	c.Assert(f.Config().Options["title"].Default, gc.Equals, "My Title")
	c.Assert(f.Actions(), jc.DeepEquals,
		&charm.Actions{
			map[string]charm.ActionSpec{
				"snapshot": {
					Description: "Take a snapshot of the database.",
					Params: map[string]interface{}{
						"type":        "object",
						"description": "Take a snapshot of the database.",
						"title":       "snapshot",
						"properties": map[string]interface{}{
							"outfile": map[string]interface{}{
								"description": "The file to write out to.",
								"type":        "string",
								"default":     "foo.bz2",
							}}}}}})
	switch f := f.(type) {
	case *charm.CharmArchive:
		c.Assert(f.Path, gc.Equals, path)
	case *charm.CharmDir:
		c.Assert(f.Path, gc.Equals, path)
	}
}

type YamlHacker map[interface{}]interface{}

func ReadYaml(r io.Reader) YamlHacker {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	m := make(map[interface{}]interface{})
	err = yaml.Unmarshal(data, m)
	if err != nil {
		panic(err)
	}
	return YamlHacker(m)
}

func (yh YamlHacker) Reader() io.Reader {
	data, err := yaml.Marshal(yh)
	if err != nil {
		panic(err)
	}
	return bytes.NewBuffer(data)
}

// charmDirPath returns the path to the charm with the
// given name in the testing repository.
func charmDirPath(c *gc.C, name string) string {
	path := filepath.Join("internal/test-charm-repo/quantal", name)
	assertIsDir(c, path)
	return path
}

// bundleDirPath returns the path to the bundle with the
// given name in the testing repository.
func bundleDirPath(c *gc.C, name string) string {
	path := filepath.Join("internal/test-charm-repo/bundle", name)
	assertIsDir(c, path)
	return path
}

func assertIsDir(c *gc.C, path string) {
	info, err := os.Stat(path)
	c.Assert(err, gc.IsNil)
	c.Assert(info.IsDir(), gc.Equals, true)
}

// readCharmDir returns the charm with the given
// name from the testing repository.
func readCharmDir(c *gc.C, name string) *charm.CharmDir {
	path := charmDirPath(c, name)
	ch, err := charm.ReadCharmDir(path)
	c.Assert(err, gc.IsNil)
	return ch
}

// readBundleDir returns the bundle with the
// given name from the testing repository.
func readBundleDir(c *gc.C, name string) *charm.BundleDir {
	path := bundleDirPath(c, name)
	ch, err := charm.ReadBundleDir(path)
	c.Assert(err, gc.IsNil)
	return ch
}

type ArchiverTo interface {
	ArchiveTo(w io.Writer) error
}

// archivePath archives the given charm or bundle
// to a newly created file and returns the path to the
// file.
func archivePath(c *gc.C, a ArchiverTo) string {
	dir := c.MkDir()
	path := filepath.Join(dir, "archive")
	file, err := os.Create(path)
	c.Assert(err, gc.IsNil)
	defer file.Close()
	err = a.ArchiveTo(file)
	c.Assert(err, gc.IsNil)
	return path
}

// cloneDir recursively copies the path directory
// into a new directory and returns the path
// to it.
func cloneDir(c *gc.C, path string) string {
	newPath := filepath.Join(c.MkDir(), filepath.Base(path))
	err := fs.Copy(path, newPath)
	c.Assert(err, gc.IsNil)
	return newPath
}

func (s *CharmSuite) assertVersionFile(c *gc.C, execName string, args []string) {
	// Read the charmDir from the testing folder
	charmDir := charmDirPath(c, "dummy")

	testing.PatchExecutableAsEchoArgs(c, s, execName)

	// copy all the contents from 'path' to 'tmp folder dummy-charm'
	// Using cloneDir
	tempPath := cloneDir(c, charmDir)

	// create an empty .execName file inside tempDir
	vcsPath := filepath.Join(tempPath, "."+execName)
	_, err := os.Create(vcsPath)
	c.Assert(err, jc.ErrorIsNil)

	err = charm.MaybeCreateVersionFile(tempPath)
	c.Assert(err, jc.ErrorIsNil)

	testing.AssertEchoArgs(c, execName, args...)

	// Verify if version exists.
	versionPath := filepath.Join(tempPath, "version")
	_, err = os.Stat(versionPath)
	c.Assert(err, jc.ErrorIsNil)

	expectedVersion := make([]string, 1, 2)
	for pos := range args {
		args[pos] = "'" + args[pos] + "'"
	}
	expectedVersion[0] = execName + " " + strings.Join(args, " ")

	f, err := os.Open(versionPath)
	c.Assert(err, jc.ErrorIsNil)
	defer f.Close()

	var version []byte
	version, err = ioutil.ReadAll(f)
	c.Assert(err, jc.ErrorIsNil)

	actualVersion := strings.TrimSuffix(string(version), "\n")

	c.Assert(actualVersion, gc.Equals, strings.Join(expectedVersion, " "))
}

// TestCreateMaybeCreateVersionFile verifies if the version file can be created
// in case of git revision control directory
func (s *CharmSuite) TestGitMaybeCreateVersionFile(c *gc.C) {

	s.assertVersionFile(c, "git", []string{"describe", "--dirty"})
}

// TestBzrMaybeCreateVersionFile verifies if the version file can be created
// in case of bazaar revision control directory.
func (s *CharmSuite) TestBazaarMaybeCreateVersionFile(c *gc.C) {
	s.assertVersionFile(c, "bzr", []string{"version-info"})
}

// TestHgMaybeCreateVersionFile verifies if the version file can be created
// in case of Mecurial revision control directory.
func (s *CharmSuite) TestHgMaybeCreateVersionFile(c *gc.C) {
	s.assertVersionFile(c, "hg", []string{"id", "-n"})
}

// TestNOVCSMaybeCreateVersionFile verifies that version file not created
// in case of not a revision control directory.
func (s *CharmSuite) TestNoVCSMaybeCreateVersionFile(c *gc.C) {
	// Read the charmDir from the testing folder.
	dummyPath := charmDirPath(c, "dummy")

	// copy all the contents from 'path' to 'tmp folder dummy-charm'.
	// Using cloneDir.
	tempPath := cloneDir(c, dummyPath)

	err := charm.MaybeCreateVersionFile(tempPath)
	c.Assert(err, gc.IsNil)

	versionPath := filepath.Join(tempPath, "version")
	_, err = os.Stat(versionPath)
	c.Assert(err, jc.Satisfies, os.IsNotExist)
}
