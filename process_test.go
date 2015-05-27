package charm_test

import (
	"strings"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v6-unstable"
)

func (s *MetaSuite) TestProcessNameRequired(c *gc.C) {
	proc := charm.Process{}
	c.Assert(proc.Validate(nil), gc.ErrorMatches, "missing name")
}

func (s *MetaSuite) TestProcesses(c *gc.C) {
	// "type" is the only required attribute for storage.
	meta, err := charm.ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
processes:
  proc0:
    type:
      name: docker
      publish_all: true
    command: foocmd
    image: nginx/nginx
    ports:
        - 80:8080
        - 443:8081
    volumes:
        - /var/www/html:/usr/share/nginx/html:ro
        - /var/nginx/conf:/etc/nginx:ro
    env:
        ENV_VAR: config:config-var
        OTHER_VAR: some value
  proc1:
    type:
      name: rkt
`))
	c.Assert(err, gc.IsNil)
	c.Assert(meta.Processes, gc.DeepEquals, map[string]charm.Process{
		"proc0": {
			Name: "proc0",
			Type: "docker",
			TypeOptions: map[string]string{
				"publish_all": "true",
			},
			Command: "foocmd",
			Image:   "nginx/nginx",
			Ports: []charm.ProcessPort{{
				External: 80,
				Internal: 8080,
			}, {
				External: 443,
				Internal: 8081,
			}},
			Volumes: []charm.ProcessVolume{{
				ExternalMount: "/var/www/html",
				InternalMount: "/usr/share/nginx/html",
				Mode:          "ro",
			}, {
				ExternalMount: "/var/nginx/conf",
				InternalMount: "/etc/nginx",
				Mode:          "ro",
			}},
			EnvVars: map[string]string{
				"ENV_VAR":   "config:config-var",
				"OTHER_VAR": "some value",
			},
		},
		"proc1": {
			Name: "proc1",
			Type: "rkt",
		},
	})
}

func (s *MetaSuite) TestProcessesNotRequired(c *gc.C) {
	noProc := strings.NewReader(`
name: a
summary: b
description: c
`)
	_, err := charm.ReadMeta(noProc)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *MetaSuite) TestProcessesTypeRequired(c *gc.C) {
	badProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  badproc:
`)
	_, err := charm.ReadMeta(badProc)
	c.Assert(err, gc.ErrorMatches, "metadata: processes.badproc: expected map, got nothing")
}

func (s *MetaSuite) TestProcessesTypeNameRequired(c *gc.C) {
	badProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  badproc:
    type:
      foo: bar
`)
	_, err := charm.ReadMeta(badProc)
	c.Assert(err, gc.ErrorMatches, "metadata: processes.badproc.type: name is required")
}

func (s *MetaSuite) TestProcessesStorageFound(c *gc.C) {
	storageProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  badproc:
    type:
      name: docker
    volumes:
      - <store0>:/var/www/html:ro
storage:
    store0:
      type: filesystem
      location: /var/lib/things
`)
	_, err := charm.ReadMeta(storageProc)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *MetaSuite) TestProcessesStorageNotFound(c *gc.C) {
	storageProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  badproc:
    type:
      name: docker
    volumes:
      - <store1>:/var/www/html:ro
storage:
    store0:
        type: filesystem
        location: /var/lib/things
`)
	_, err := charm.ReadMeta(storageProc)
	c.Assert(err, gc.ErrorMatches, "metadata: processes.badproc.volumes: specified storage \"store1\" unknown for .*")
}
