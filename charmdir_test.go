// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/juju/testing"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable"
)

type CharmDirSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&CharmDirSuite{})

func (s *CharmDirSuite) TestReadCharmDir(c *gc.C) {
	path := charmDirPath(c, "dummy")
	dir, err := charm.ReadCharmDir(path)
	c.Assert(err, gc.IsNil)
	checkDummy(c, dir, path)
}

func (s *CharmDirSuite) TestReadCharmDirWithoutConfig(c *gc.C) {
	path := charmDirPath(c, "varnish")
	dir, err := charm.ReadCharmDir(path)
	c.Assert(err, gc.IsNil)

	// A lacking config.yaml file still causes a proper
	// Config value to be returned.
	c.Assert(dir.Config().Options, gc.HasLen, 0)
}

func (s *CharmDirSuite) TestReadCharmDirWithoutMetrics(c *gc.C) {
	path := charmDirPath(c, "varnish")
	dir, err := charm.ReadCharmDir(path)
	c.Assert(err, gc.IsNil)

	// A lacking metrics.yaml file indicates the unit will not
	// be metered.
	c.Assert(dir.Metrics(), gc.IsNil)
}

func (s *CharmDirSuite) TestReadCharmDirWithEmptyMetrics(c *gc.C) {
	path := charmDirPath(c, "metered-empty")
	dir, err := charm.ReadCharmDir(path)
	c.Assert(err, gc.IsNil)
	c.Assert(Keys(dir.Metrics()), gc.HasLen, 0)
}

func (s *CharmDirSuite) TestReadCharmDirWithCustomMetrics(c *gc.C) {
	path := charmDirPath(c, "metered")
	dir, err := charm.ReadCharmDir(path)
	c.Assert(err, gc.IsNil)

	c.Assert(dir.Metrics(), gc.NotNil)
	c.Assert(Keys(dir.Metrics()), gc.DeepEquals, []string{"juju-unit-time", "pings"})
}

func (s *CharmDirSuite) TestReadCharmDirWithoutActions(c *gc.C) {
	path := charmDirPath(c, "wordpress")
	dir, err := charm.ReadCharmDir(path)
	c.Assert(err, gc.IsNil)

	// A lacking actions.yaml file still causes a proper
	// Actions value to be returned.
	c.Assert(dir.Actions().ActionSpecs, gc.HasLen, 0)
}

func (s *CharmDirSuite) TestArchiveTo(c *gc.C) {
	baseDir := c.MkDir()
	charmDir := cloneDir(c, charmDirPath(c, "dummy"))
	s.assertArchiveTo(c, baseDir, charmDir)
}

func (s *CharmDirSuite) TestArchiveToWithSymlinkedRootDir(c *gc.C) {
	path := cloneDir(c, charmDirPath(c, "dummy"))
	baseDir := filepath.Dir(path)
	err := os.Symlink(filepath.Join("dummy"), filepath.Join(baseDir, "newdummy"))
	c.Assert(err, gc.IsNil)
	charmDir := filepath.Join(baseDir, "newdummy")

	s.assertArchiveTo(c, baseDir, charmDir)
}

func (s *CharmDirSuite) assertArchiveTo(c *gc.C, baseDir, charmDir string) {
	haveSymlinks := true
	if err := os.Symlink("../target", filepath.Join(charmDir, "hooks/symlink")); err != nil {
		haveSymlinks = false
	}
	dir, err := charm.ReadCharmDir(charmDir)
	c.Assert(err, gc.IsNil)
	path := filepath.Join(baseDir, "archive.charm")
	file, err := os.Create(path)
	c.Assert(err, gc.IsNil)
	err = dir.ArchiveTo(file)
	file.Close()
	c.Assert(err, gc.IsNil)

	zipr, err := zip.OpenReader(path)
	c.Assert(err, gc.IsNil)
	defer zipr.Close()

	var metaf, instf, emptyf, revf, symf *zip.File
	for _, f := range zipr.File {
		c.Logf("Archived file: %s", f.Name)
		switch f.Name {
		case "revision":
			revf = f
		case "metadata.yaml":
			metaf = f
		case "hooks/install":
			instf = f
		case "hooks/symlink":
			symf = f
		case "empty/":
			emptyf = f
		case "build/ignored":
			c.Errorf("archive includes build/*: %s", f.Name)
		case ".ignored", ".dir/ignored":
			c.Errorf("archive includes .* entries: %s", f.Name)
		}
	}

	c.Assert(revf, gc.NotNil)
	reader, err := revf.Open()
	c.Assert(err, gc.IsNil)
	data, err := ioutil.ReadAll(reader)
	reader.Close()
	c.Assert(err, gc.IsNil)
	c.Assert(string(data), gc.Equals, "1")

	c.Assert(metaf, gc.NotNil)
	reader, err = metaf.Open()
	c.Assert(err, gc.IsNil)
	meta, err := charm.ReadMeta(reader)
	reader.Close()
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Name, gc.Equals, "dummy")

	c.Assert(instf, gc.NotNil)
	// Despite it being 0751, we pack and unpack it as 0755.
	c.Assert(instf.Mode()&0777, gc.Equals, os.FileMode(0755))

	if haveSymlinks {
		c.Assert(symf, gc.NotNil)
		c.Assert(symf.Mode()&0777, gc.Equals, os.FileMode(0777))
		reader, err = symf.Open()
		c.Assert(err, gc.IsNil)
		data, err = ioutil.ReadAll(reader)
		reader.Close()
		c.Assert(err, gc.IsNil)
		c.Assert(string(data), gc.Equals, "../target")
	} else {
		c.Assert(symf, gc.IsNil)
	}

	c.Assert(emptyf, gc.NotNil)
	c.Assert(emptyf.Mode()&os.ModeType, gc.Equals, os.ModeDir)
	// Despite it being 0750, we pack and unpack it as 0755.
	c.Assert(emptyf.Mode()&0777, gc.Equals, os.FileMode(0755))
}

// Bug #864164: Must complain if charm hooks aren't executable
func (s *CharmDirSuite) TestArchiveToWithNonExecutableHooks(c *gc.C) {
	hooks := []string{"install", "start", "config-changed", "upgrade-charm", "stop", "collect-metrics", "meter-status-changed"}
	for _, relName := range []string{"foo", "bar", "self"} {
		for _, kind := range []string{"joined", "changed", "departed", "broken"} {
			hooks = append(hooks, relName+"-relation-"+kind)
		}
	}

	dir := readCharmDir(c, "all-hooks")
	path := filepath.Join(c.MkDir(), "archive.charm")
	file, err := os.Create(path)
	c.Assert(err, gc.IsNil)
	err = dir.ArchiveTo(file)
	file.Close()
	c.Assert(err, gc.IsNil)

	tlog := c.GetTestLog()
	for _, hook := range hooks {
		fullpath := filepath.Join(dir.Path, "hooks", hook)
		exp := fmt.Sprintf(`^(.|\n)*WARNING juju.charm making "%s" executable in charm(.|\n)*$`, fullpath)
		c.Assert(tlog, gc.Matches, exp, gc.Commentf("hook %q was not made executable", fullpath))
	}

	// Expand it and check the hooks' permissions
	// (But do not use ExpandTo(), just use the raw zip)
	f, err := os.Open(path)
	c.Assert(err, gc.IsNil)
	defer f.Close()
	fi, err := f.Stat()
	c.Assert(err, gc.IsNil)
	size := fi.Size()
	zipr, err := zip.NewReader(f, size)
	c.Assert(err, gc.IsNil)
	allhooks := dir.Meta().Hooks()
	for _, zfile := range zipr.File {
		cleanName := filepath.Clean(zfile.Name)
		if strings.HasPrefix(cleanName, "hooks") {
			hookName := filepath.Base(cleanName)
			if _, ok := allhooks[hookName]; ok {
				perms := zfile.Mode()
				c.Assert(perms&0100 != 0, gc.Equals, true, gc.Commentf("hook %q is not executable", hookName))
			}
		}
	}
}

func (s *CharmDirSuite) TestArchiveToWithBadType(c *gc.C) {
	charmDir := cloneDir(c, charmDirPath(c, "dummy"))
	badFile := filepath.Join(charmDir, "hooks", "badfile")

	// Symlink targeting a path outside of the charm.
	err := os.Symlink("../../target", badFile)
	c.Assert(err, gc.IsNil)

	dir, err := charm.ReadCharmDir(charmDir)
	c.Assert(err, gc.IsNil)

	err = dir.ArchiveTo(&bytes.Buffer{})
	c.Assert(err, gc.ErrorMatches, `symlink "hooks/badfile" links out of charm: "../../target"`)

	// Symlink targeting an absolute path.
	os.Remove(badFile)
	err = os.Symlink("/target", badFile)
	c.Assert(err, gc.IsNil)

	dir, err = charm.ReadCharmDir(charmDir)
	c.Assert(err, gc.IsNil)

	err = dir.ArchiveTo(&bytes.Buffer{})
	c.Assert(err, gc.ErrorMatches, `symlink "hooks/badfile" is absolute: "/target"`)

	// Can't archive special files either.
	os.Remove(badFile)
	err = syscall.Mkfifo(badFile, 0644)
	c.Assert(err, gc.IsNil)

	dir, err = charm.ReadCharmDir(charmDir)
	c.Assert(err, gc.IsNil)

	err = dir.ArchiveTo(&bytes.Buffer{})
	c.Assert(err, gc.ErrorMatches, `file is a named pipe: "hooks/badfile"`)
}

func (s *CharmDirSuite) TestDirRevisionFile(c *gc.C) {
	charmDir := cloneDir(c, charmDirPath(c, "dummy"))
	revPath := filepath.Join(charmDir, "revision")

	// Missing revision file
	err := os.Remove(revPath)
	c.Assert(err, gc.IsNil)

	dir, err := charm.ReadCharmDir(charmDir)
	c.Assert(err, gc.IsNil)
	c.Assert(dir.Revision(), gc.Equals, 0)

	// Missing revision file with old revision in metadata
	file, err := os.OpenFile(filepath.Join(charmDir, "metadata.yaml"), os.O_WRONLY|os.O_APPEND, 0)
	c.Assert(err, gc.IsNil)
	_, err = file.Write([]byte("\nrevision: 1234\n"))
	c.Assert(err, gc.IsNil)

	dir, err = charm.ReadCharmDir(charmDir)
	c.Assert(err, gc.IsNil)
	c.Assert(dir.Revision(), gc.Equals, 1234)

	// Revision file with bad content
	err = ioutil.WriteFile(revPath, []byte("garbage"), 0666)
	c.Assert(err, gc.IsNil)

	dir, err = charm.ReadCharmDir(charmDir)
	c.Assert(err, gc.ErrorMatches, "invalid revision file")
	c.Assert(dir, gc.IsNil)
}

func (s *CharmDirSuite) TestDirSetRevision(c *gc.C) {
	path := cloneDir(c, charmDirPath(c, "dummy"))
	dir, err := charm.ReadCharmDir(path)
	c.Assert(err, gc.IsNil)
	c.Assert(dir.Revision(), gc.Equals, 1)
	dir.SetRevision(42)
	c.Assert(dir.Revision(), gc.Equals, 42)

	var b bytes.Buffer
	err = dir.ArchiveTo(&b)
	c.Assert(err, gc.IsNil)

	archive, err := charm.ReadCharmArchiveBytes(b.Bytes())
	c.Assert(archive.Revision(), gc.Equals, 42)
}

func (s *CharmDirSuite) TestDirSetDiskRevision(c *gc.C) {
	charmDir := cloneDir(c, charmDirPath(c, "dummy"))
	dir, err := charm.ReadCharmDir(charmDir)
	c.Assert(err, gc.IsNil)

	c.Assert(dir.Revision(), gc.Equals, 1)
	dir.SetDiskRevision(42)
	c.Assert(dir.Revision(), gc.Equals, 42)

	dir, err = charm.ReadCharmDir(charmDir)
	c.Assert(err, gc.IsNil)
	c.Assert(dir.Revision(), gc.Equals, 42)
}
