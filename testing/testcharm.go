// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/juju/testing/filetesting"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v4"
)

// Charm holds a charm for testing. It does not
// have a representation on disk by default, but
// can be written to disk using Archive and its ExpandTo
// method. It implements the charm.Charm interface.
//
// All methods on Charm may be called concurrently.
type Charm struct {
	meta     *charm.Meta
	config   *charm.Config
	actions  *charm.Actions
	metrics  *charm.Metrics
	revision int

	files filetesting.Entries

	makeArchiveOnce sync.Once
	archiveBytes    []byte
	archive         *charm.CharmArchive
}

// CharmSpec holds the specification for a charm. The fields
// hold data in YAML format.
type CharmSpec struct {
	// Meta holds the contents of metadata.yaml.
	Meta string

	// Config holds the contents of config.yaml.
	Config string

	// Actions holds the contents of actions.yaml.
	Actions string

	// Metrics holds the contents of metrics.yaml.
	Metrics string

	// Files holds any additional files that should be
	// added to the charm. If this is nil, a minimal set
	// of files will be added to ensure the charm is readable.
	Files []filetesting.Entry

	// Revision specifies the revision of the charm.
	Revision int
}

// NewCharm returns a new charm
func NewCharm(c *gc.C, spec CharmSpec) *Charm {
	ch := &Charm{
		revision: spec.Revision,
	}
	var err error
	ch.meta, err = charm.ReadMeta(strings.NewReader(spec.Meta))
	c.Assert(err, gc.IsNil)
	ch.files = append(ch.files, filetesting.File{
		Path: "metadata.yaml",
		Data: spec.Meta,
		Perm: 0644,
	})

	if spec.Config != "" {
		ch.config, err = charm.ReadConfig(strings.NewReader(spec.Config))
		c.Assert(err, gc.IsNil)
		ch.files = append(ch.files, filetesting.File{
			Path: "config.yaml",
			Data: spec.Config,
			Perm: 0644,
		})
	}
	if spec.Actions != "" {
		ch.actions, err = charm.ReadActionsYaml(strings.NewReader(spec.Actions))
		c.Assert(err, gc.IsNil)
		ch.files = append(ch.files, filetesting.File{
			Path: "actions.yaml",
			Data: spec.Actions,
			Perm: 0644,
		})
	}
	if spec.Metrics != "" {
		ch.metrics, err = charm.ReadMetrics(strings.NewReader(spec.Metrics))
		c.Assert(err, gc.IsNil)
		ch.files = append(ch.files, filetesting.File{
			Path: "metrics.yaml",
			Data: spec.Metrics,
			Perm: 0644,
		})
	}
	if spec.Files == nil {
		ch.files = append(ch.files, filetesting.File{
			Path: "hooks/install",
			Data: "#!/bin/sh\n",
			Perm: 0755,
		}, filetesting.File{
			Path: "hooks/start",
			Data: "#!/bin/sh\n",
			Perm: 0755,
		})
	} else {
		ch.files = append(ch.files, spec.Files...)
		// Check for duplicates.
		names := make(map[string]bool)
		for _, f := range ch.files {
			name := path.Clean(f.GetPath())
			if names[name] {
				panic(fmt.Errorf("duplicate file entry %q", f.GetPath()))
			}
			names[name] = true
		}
	}
	return ch
}

// Meta implements charm.Charm.Meta.
func (ch *Charm) Meta() *charm.Meta {
	return ch.meta
}

// Config implements charm.Charm.Config.
func (ch *Charm) Config() *charm.Config {
	if ch.config == nil {
		return &charm.Config{
			Options: map[string]charm.Option{},
		}
	}
	return ch.config
}

// Metrics implements charm.Charm.Metrics.
func (ch *Charm) Metrics() *charm.Metrics {
	return ch.metrics
}

// Actions implements charm.Charm.Actions.
func (ch *Charm) Actions() *charm.Actions {
	if ch.actions == nil {
		return &charm.Actions{}
	}
	return ch.actions
}

// Revision implements charm.Charm.Revision.
func (ch *Charm) Revision() int {
	return ch.revision
}

// Archive returns a charm archive holding the charm.
func (ch *Charm) Archive() *charm.CharmArchive {
	ch.makeArchiveOnce.Do(ch.makeArchive)
	return ch.archive
}

// ArchiveBytes returns the contents of the charm archive
// holding the charm.
func (ch *Charm) ArchiveBytes() []byte {
	ch.makeArchiveOnce.Do(ch.makeArchive)
	return ch.archiveBytes
}

func (ch *Charm) makeArchive() {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, f := range ch.files {
		addZipEntry(zw, f)
	}
	if err := zw.Close(); err != nil {
		panic(err)
	}
	// ReadCharmArchiveFromReader requires a ReaderAt, so make one.
	r := bytes.NewReader(buf.Bytes())

	// Actually make the charm archive.
	archive, err := charm.ReadCharmArchiveFromReader(r, int64(buf.Len()))
	if err != nil {
		panic(err)
	}
	ch.archiveBytes = buf.Bytes()
	ch.archive = archive
	ch.archive.SetRevision(ch.revision)
}

func addZipEntry(zw *zip.Writer, f filetesting.Entry) {
	h := &zip.FileHeader{
		Name: f.GetPath(),
		// Don't bother compressing - the contents are so small that
		// it will just slow things down for no particular benefit.
		Method: zip.Store,
	}
	contents := ""
	switch f := f.(type) {
	case filetesting.Dir:
		h.SetMode(os.ModeDir | 0755)
	case filetesting.File:
		h.SetMode(f.Perm)
		contents = f.Data
	case filetesting.Symlink:
		h.SetMode(os.ModeSymlink | 0777)
		contents = f.Link
	}
	w, err := zw.CreateHeader(h)
	if err != nil {
		panic(err)
	}
	if contents != "" {
		if _, err := w.Write([]byte(contents)); err != nil {
			panic(err)
		}
	}
}
