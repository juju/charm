// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/juju/errors"
)

// defaultJujuIgnore contains jujuignore directives for excluding VCS- and
// build-related directories when archiving. The following set of directives
// will be prepended to the contents of the charm's .jujuignore file if one is
// provided.
//
// NOTE: writeArchive auto-generates its own revision and version files so they
// need to be excluded here to prevent anyone from overriding their contents by
// adding files with the same name to their charm repo.
var defaultJujuIgnore = `
.git
.svn
.hg
.bzr
.tox

/build/
/revision
/version

.jujuignore
`

// The CharmDir type encapsulates access to data and operations
// on a charm directory.
type CharmDir struct {
	Path       string
	meta       *Meta
	config     *Config
	metrics    *Metrics
	actions    *Actions
	lxdProfile *LXDProfile
	revision   int
	version    string
}

// Trick to ensure *CharmDir implements the Charm interface.
var _ Charm = (*CharmDir)(nil)

// IsCharmDir report whether the path is likely to represent
// a charm, even it may be incomplete.
func IsCharmDir(path string) bool {
	dir := &CharmDir{Path: path}
	_, err := os.Stat(dir.join("metadata.yaml"))
	return err == nil
}

// ReadCharmDir returns a CharmDir representing an expanded charm directory.
func ReadCharmDir(path string) (dir *CharmDir, err error) {
	dir = &CharmDir{Path: path}
	file, err := os.Open(dir.join("metadata.yaml"))
	if err != nil {
		return nil, err
	}
	dir.meta, err = ReadMeta(file)
	file.Close()
	if err != nil {
		return nil, err
	}

	file, err = os.Open(dir.join("config.yaml"))
	if _, ok := err.(*os.PathError); ok {
		dir.config = NewConfig()
	} else if err != nil {
		return nil, err
	} else {
		dir.config, err = ReadConfig(file)
		file.Close()
		if err != nil {
			return nil, err
		}
	}

	file, err = os.Open(dir.join("metrics.yaml"))
	if err == nil {
		dir.metrics, err = ReadMetrics(file)
		file.Close()
		if err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	file, err = os.Open(dir.join("actions.yaml"))
	if _, ok := err.(*os.PathError); ok {
		dir.actions = NewActions()
	} else if err != nil {
		return nil, err
	} else {
		dir.actions, err = ReadActionsYaml(file)
		file.Close()
		if err != nil {
			return nil, err
		}
	}

	if file, err = os.Open(dir.join("revision")); err == nil {
		_, err = fmt.Fscan(file, &dir.revision)
		file.Close()
		if err != nil {
			return nil, errors.New("invalid revision file")
		}
	}

	var dummyLogger = NopLogger{}
	version, _, _ := dir.MaybeGenerateVersionString(dummyLogger)
	dir.version = version

	file, err = os.Open(dir.join("lxd-profile.yaml"))
	if _, ok := err.(*os.PathError); ok {
		dir.lxdProfile = NewLXDProfile()
	} else if err != nil {
		return nil, err
	} else {
		dir.lxdProfile, err = ReadLXDProfile(file)
		file.Close()
		if err != nil {
			return nil, err
		}
	}

	return dir, nil
}

// buildIgnoreRules parses the contents of the charm's .jujuignore file and
// compiles a set of rules that are used to decide which files should be
// archived.
func (dir *CharmDir) buildIgnoreRules() (ignoreRuleset, error) {
	// Start with a set of sane defaults to ensure backwards-compatibility
	// for charms that do not use a .jujuignore file.
	rules, err := newIgnoreRuleset(strings.NewReader(defaultJujuIgnore))
	if err != nil {
		return nil, err
	}

	pathToJujuignore := dir.join(".jujuignore")
	if _, err := os.Stat(pathToJujuignore); err == nil {
		file, err := os.Open(dir.join(".jujuignore"))
		if err != nil {
			return nil, err
		}
		defer func() { _ = file.Close() }()

		jujuignoreRules, err := newIgnoreRuleset(file)
		if err != nil {
			return nil, errors.Annotate(err, ".jujuignore")
		}

		rules = append(rules, jujuignoreRules...)
	}

	return rules, nil
}

// join builds a path rooted at the charm's expanded directory
// path and the extra path components provided.
func (dir *CharmDir) join(parts ...string) string {
	parts = append([]string{dir.Path}, parts...)
	return filepath.Join(parts...)
}

// Revision returns the revision number for the charm
// expanded in dir.
func (dir *CharmDir) Revision() int {
	return dir.revision
}

// Version returns the VCS version representing the version file from archive.
func (dir *CharmDir) Version() string {
	return dir.version
}

// Meta returns the Meta representing the metadata.yaml file
// for the charm expanded in dir.
func (dir *CharmDir) Meta() *Meta {
	return dir.meta
}

// Config returns the Config representing the config.yaml file
// for the charm expanded in dir.
func (dir *CharmDir) Config() *Config {
	return dir.config
}

// Metrics returns the Metrics representing the metrics.yaml file
// for the charm expanded in dir.
func (dir *CharmDir) Metrics() *Metrics {
	return dir.metrics
}

// Actions returns the Actions representing the actions.yaml file
// for the charm expanded in dir.
func (dir *CharmDir) Actions() *Actions {
	return dir.actions
}

// LXDProfile returns the LXDProfile representing the lxd-profile.yaml file
// for the charm expanded in dir.
func (dir *CharmDir) LXDProfile() *LXDProfile {
	return dir.lxdProfile
}

// SetRevision changes the charm revision number. This affects
// the revision reported by Revision and the revision of the
// charm archived by ArchiveTo.
// The revision file in the charm directory is not modified.
func (dir *CharmDir) SetRevision(revision int) {
	dir.revision = revision
}

// SetDiskRevision does the same as SetRevision but also changes
// the revision file in the charm directory.
func (dir *CharmDir) SetDiskRevision(revision int) error {
	dir.SetRevision(revision)
	file, err := os.OpenFile(dir.join("revision"), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(strconv.Itoa(revision)))
	file.Close()
	return err
}

// resolveSymlinkedRoot returns the target destination of a
// charm root directory if the root directory is a symlink.
func resolveSymlinkedRoot(rootPath string) (string, error) {
	info, err := os.Lstat(rootPath)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		rootPath, err = filepath.EvalSymlinks(rootPath)
		if err != nil {
			return "", fmt.Errorf("cannot read path symlink at %q: %v", rootPath, err)
		}
	}
	return rootPath, nil
}

// ArchiveTo creates a charm file from the charm expanded in dir.
// By convention a charm archive should have a ".charm" suffix.
func (dir *CharmDir) ArchiveTo(w io.Writer) error {
	ignoreRules, err := dir.buildIgnoreRules()
	if err != nil {
		return err
	}
	// We update the version to make sure we don't lag behind
	dir.version, _, err = dir.MaybeGenerateVersionString(logger)
	if err != nil {
		// We don't want to stop, even if the version cannot be generated
		logger.Warningf("%v", err)
	}

	return writeArchive(w, dir.Path, dir.revision, dir.version, dir.Meta().Hooks(), ignoreRules)
}

func writeArchive(w io.Writer, path string, revision int, versionString string, hooks map[string]bool, ignoreRules ignoreRuleset) error {
	zipw := zip.NewWriter(w)
	defer zipw.Close()

	// The root directory may be symlinked elsewhere so
	// resolve that before creating the zip.
	rootPath, err := resolveSymlinkedRoot(path)
	if err != nil {
		return err
	}
	zp := zipPacker{zipw, rootPath, hooks, ignoreRules}
	if revision != -1 {
		zp.AddFile("revision", strconv.Itoa(revision))
	}
	if versionString != "" {
		zp.AddFile("version", versionString)
	}
	return filepath.Walk(rootPath, zp.WalkFunc())
}

type zipPacker struct {
	*zip.Writer
	root        string
	hooks       map[string]bool
	ignoreRules ignoreRuleset
}

func (zp *zipPacker) WalkFunc() filepath.WalkFunc {
	return func(path string, fi os.FileInfo, err error) error {
		return zp.visit(path, fi, err)
	}
}

func (zp *zipPacker) AddFile(filename string, value string) error {
	h := &zip.FileHeader{Name: filename}
	h.SetMode(syscall.S_IFREG | 0644)
	w, err := zp.CreateHeader(h)
	if err == nil {
		_, err = w.Write([]byte(value))
	}
	return err
}

func (zp *zipPacker) visit(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	relpath, err := filepath.Rel(zp.root, path)
	if err != nil {
		return err
	}

	// Replace any Windows path separators with "/".
	// zip file spec 4.4.17.1 says that separators are always "/" even on Windows.
	relpath = filepath.ToSlash(relpath)

	// Check if this file or dir needs to be ignored
	if zp.ignoreRules.Match(relpath, fi.IsDir()) {
		if fi.IsDir() {
			return filepath.SkipDir
		}

		return nil
	}

	method := zip.Deflate
	if fi.IsDir() {
		relpath += "/"
		method = zip.Store
	}

	mode := fi.Mode()
	if err := checkFileType(relpath, mode); err != nil {
		return err
	}
	if mode&os.ModeSymlink != 0 {
		method = zip.Store
	}
	h := &zip.FileHeader{
		Name:   relpath,
		Method: method,
	}

	perm := os.FileMode(0644)
	if mode&os.ModeSymlink != 0 {
		perm = 0777
	} else if mode&0100 != 0 {
		perm = 0755
	}
	if filepath.Dir(relpath) == "hooks" {
		hookName := filepath.Base(relpath)
		if _, ok := zp.hooks[hookName]; ok && !fi.IsDir() && mode&0100 == 0 {
			logger.Warningf("making %q executable in charm", path)
			perm = perm | 0100
		}
	}
	h.SetMode(mode&^0777 | perm)

	w, err := zp.CreateHeader(h)
	if err != nil || fi.IsDir() {
		return err
	}
	var data []byte
	if mode&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}
		if err := checkSymlinkTarget(zp.root, relpath, target); err != nil {
			return err
		}
		data = []byte(target)
		_, err = w.Write(data)
	} else {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(w, file)
	}
	return err
}

func checkSymlinkTarget(basedir, symlink, target string) error {
	if filepath.IsAbs(target) {
		return fmt.Errorf("symlink %q is absolute: %q", symlink, target)
	}
	p := filepath.Join(filepath.Dir(symlink), target)
	if p == ".." || strings.HasPrefix(p, "../") {
		return fmt.Errorf("symlink %q links out of charm: %q", symlink, target)
	}
	return nil
}

func checkFileType(path string, mode os.FileMode) error {
	e := "file has an unknown type: %q"
	switch mode & os.ModeType {
	case os.ModeDir, os.ModeSymlink, 0:
		return nil
	case os.ModeNamedPipe:
		e = "file is a named pipe: %q"
	case os.ModeSocket:
		e = "file is a socket: %q"
	case os.ModeDevice:
		e = "file is a device: %q"
	}
	return fmt.Errorf(e, path)
}

// Logger represents the logging methods called.
type Logger interface {
	Warningf(message string, args ...interface{})
	Debugf(message string, args ...interface{})
	Errorf(message string, args ...interface{})
	Tracef(message string, args ...interface{})
	Infof(message string, args ...interface{})
}

// NopLogger is used to stop our logger from logging
type NopLogger struct {
}

func (m NopLogger) Warningf(message string, args ...interface{}) {

}

func (m NopLogger) Debugf(message string, args ...interface{}) {

}

func (m NopLogger) Errorf(message string, args ...interface{}) {

}

func (m NopLogger) Tracef(message string, args ...interface{}) {

}

func (m NopLogger) Infof(message string, args ...interface{}) {

}

type vcsCMD struct {
	vcsType       string
	args          []string
	usesTypeCheck func(charmPath string) bool
}

func (v *vcsCMD) commonErrHandler(err error, charmPath string) error {
	return errors.Errorf("%q version string generation failed : "+
		"%v\nThis means that the charm version won't show in juju status. Charm path %q", v.vcsType, err, charmPath)
}

// The first check checks for the easy case of the current charmdir has a git folder.
// There can be cases when the charmdir actually uses git and is just a subdir, thus the below check
func usesGit(charmPath string) bool {
	if _, err := os.Stat(filepath.Join(charmPath, ".git")); err == nil {
		return true
	}
	args := []string{"rev-parse", "--is-inside-work-tree"}
	execCmd := exec.Command("git", args...)
	execCmd.Dir = charmPath
	if out, err := execCmd.Output(); err == nil {
		logger.Errorf("%q", out)
		return true
	}
	return false
}

func usesBzr(charmPath string) bool {
	if _, err := os.Stat(filepath.Join(charmPath, ".bzr")); err == nil {
		return true
	}
	return false
}

func usesHg(charmPath string) bool {
	if _, err := os.Stat(filepath.Join(charmPath, ".hg")); err == nil {
		return true
	}
	return false
}

// MaybeGenerateVersionString generates charm version string.
// We want to know whether parent folders use one of these vcs, that's why we try to execute each one of them
// The second return value is the detected vcs type.
func (dir *CharmDir) MaybeGenerateVersionString(logger Logger) (string, string, error) {
	vcsStrategies := make(map[string]vcsCMD)

	versionFileVersionType := "versionFile"
	mercurialStrategy := vcsCMD{"hg", []string{"id", "-n"}, usesHg}
	bazaarStrategy := vcsCMD{"bzr", []string{"version-info"}, usesBzr}
	gitStrategy := vcsCMD{"git", []string{"describe", "--dirty", "--always"}, usesGit}

	vcsStrategies["hg"] = mercurialStrategy
	vcsStrategies["git"] = gitStrategy
	vcsStrategies["bzr"] = bazaarStrategy

	// Nowadays most vcs used are git, we want to make sure that git is the first one we test
	vcsOrder := [...]string{"git", "hg", "bzr"}

	for _, vcsType := range vcsOrder {
		vcsCmd := vcsStrategies[vcsType]
		if vcsCmd.usesTypeCheck(dir.Path) {
			cmd := exec.Command(vcsCmd.vcsType, vcsCmd.args...)
			// We need to make sure that the working directory will be the one we execute the commands from.
			cmd.Dir = dir.Path
			// Version string value is written to stdout if successful.
			out, err := cmd.Output()
			if err != nil {
				// We had an error but we still know that we use a vcs thus we can stop here and handle it.
				return "", vcsType, vcsCmd.commonErrHandler(err, dir.Path)
			}
			output := string(out)
			return output, vcsType, nil
		}
	}

	// If all strategies fail we fallback to check the version below
	if file, err := os.Open(dir.join("version")); err == nil {
		logger.Debugf("charm is not in version control, but uses a version file, charm path %q", dir.Path)
		var versionNumber string
		n, err := fmt.Fscan(file, &versionNumber)
		if n != 1 {
			return "", versionFileVersionType, errors.Errorf("invalid version file, charm path: %q", dir.Path)
		}
		if err != nil {
			return "", versionFileVersionType, err
		}
		if err = file.Close(); err != nil {
			return "", versionFileVersionType, err
		}
		return versionNumber, versionFileVersionType, nil
	} else {
		logger.Infof("charm is not versioned, charm path %q", dir.Path)
		return "", "", nil
	}
}
