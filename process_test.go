package charm_test

import (
	"strings"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/yaml.v1"

	"gopkg.in/juju/charm.v6-unstable"
)

func (s *MetaSuite) TestProcessParse(c *gc.C) {
	raw := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(`
description: a process
type: docker
type-options:
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
`), raw)
	c.Assert(err, jc.ErrorIsNil)
	proc, err := charm.ParseProcess("proc0", raw)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(proc, gc.DeepEquals, &charm.Process{
		Name:        "proc0",
		Description: "a process",
		Type:        "docker",
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
	})
}

func (s *MetaSuite) TestProcessCopyVolume(c *gc.C) {
	vol := charm.ProcessVolume{
		ExternalMount: "a",
		InternalMount: "b",
		Mode:          "ro",
		Name:          "spam",
	}
	copied := vol.Copy()

	c.Check(copied, jc.DeepEquals, vol)
}

func (s *MetaSuite) TestProcessCopyProcess(c *gc.C) {
	proc := charm.Process{
		Name:        "proc0",
		Description: "a process",
		Type:        "docker",
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
	}
	copied := proc.Copy()

	c.Check(copied, jc.DeepEquals, proc)
}

func (s *MetaSuite) TestProcessApplyOkay(c *gc.C) {
	proc := &charm.Process{
		Name: "a proc",
		Type: "docker",
		TypeOptions: map[string]string{
			"publish_all": "true",
		},
		Image: "nginx/nginx-2",
		Ports: []charm.ProcessPort{{
			External: 81,
			Internal: 8001,
		}},
		Volumes: []charm.ProcessVolume{{
			ExternalMount: "/var/www/html",
			InternalMount: "/usr/share/nginx/html",
			Mode:          "rw",
		}},
		EnvVars: map[string]string{
			"ENV_VAR": "spam",
		},
	}
	overrides := []charm.ProcessFieldValue{{
		Field:    "type-options",
		Subfield: "publish_all",
		Value:    "NO",
	}, {
		Field: "image",
		Value: "nginx/nginx",
	}, {
		Field:    "ports",
		Subfield: "0",
		Value:    "80:8080",
	}, {
		Field:    "volumes",
		Subfield: "0",
		Value:    "/var/www/html:/usr/share/nginx/html:ro",
	}, {
		Field:    "env",
		Subfield: "ENV_VAR",
		Value:    "config:config-var",
	}}
	additions := []charm.ProcessFieldValue{{
		Field: "description",
		Value: "my proc",
	}, {
		Field: "command",
		Value: "foocmd",
	}, {
		Field: "ports",
		Value: "443:8081",
	}, {
		Field: "volumes",
		Value: "/var/nginx/conf:/etc/nginx:ro",
	}, {
		Field:    "env",
		Subfield: "OTHER_VAR",
		Value:    "some value",
	}}
	applied, err := proc.Apply(overrides, additions)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(applied, jc.DeepEquals, &charm.Process{
		Name:        "a proc",
		Type:        "docker",
		Description: "my proc",
		TypeOptions: map[string]string{
			"publish_all": "NO",
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
	})
}

func (s *MetaSuite) TestProcessApplyEmpty(c *gc.C) {
	proc := &charm.Process{}
	var overrides []charm.ProcessFieldValue
	additions := []charm.ProcessFieldValue{{
		Field:    "type-options",
		Subfield: "publish_all",
		Value:    "NO",
	}, {
		Field: "description",
		Value: "my proc",
	}, {
		Field: "image",
		Value: "nginx/nginx",
	}, {
		Field: "command",
		Value: "foocmd",
	}, {
		Field: "ports",
		Value: "80:8080",
	}, {
		Field: "ports",
		Value: "443:8081",
	}, {
		Field: "volumes",
		Value: "/var/www/html:/usr/share/nginx/html:ro",
	}, {
		Field: "volumes",
		Value: "/var/nginx/conf:/etc/nginx:ro",
	}, {
		Field:    "env",
		Subfield: "ENV_VAR",
		Value:    "config:config-var",
	}, {
		Field:    "env",
		Subfield: "OTHER_VAR",
		Value:    "some value",
	}}
	applied, err := proc.Apply(overrides, additions)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(applied, jc.DeepEquals, &charm.Process{
		Description: "my proc",
		TypeOptions: map[string]string{
			"publish_all": "NO",
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
	})
}

func (s *MetaSuite) TestProcessApplyNoChange(c *gc.C) {
	proc := &charm.Process{
		Name:        "a proc",
		Type:        "docker",
		Description: "my proc",
		TypeOptions: map[string]string{
			"publish_all": "NO",
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
	}
	var overrides, additions []charm.ProcessFieldValue
	applied, err := proc.Apply(overrides, additions)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(applied, jc.DeepEquals, &charm.Process{
		Name:        "a proc",
		Type:        "docker",
		Description: "my proc",
		TypeOptions: map[string]string{
			"publish_all": "NO",
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
	})
}

type procTest struct {
	desc     string
	field    string
	subfield string
	value    string
	err      string
}

func (t procTest) log(c *gc.C, i int) {
	c.Logf("test %d: %s", i, t.desc)
}

func (t procTest) changes() []charm.ProcessFieldValue {
	return []charm.ProcessFieldValue{{
		Field:    t.field,
		Subfield: t.subfield,
		Value:    t.value,
	}}
}

func (s *MetaSuite) TestProcessApplyBadOverride(c *gc.C) {
	tests := []procTest{{
		desc:  "unknown field",
		field: "spam",
		err:   "unrecognized field.*",
	}, {
		desc:  "name",
		field: "name",
		err:   "cannot override.*",
	}, {
		desc:  "type",
		field: "type",
		err:   "cannot override.*",
	}, {
		desc:  "simple field not set",
		field: "description",
		err:   "cannot override.*, not set",
	}, {
		desc:  "map missing subfield",
		field: "env",
		err:   "cannot override.* without sub-field",
	}, {
		desc:     "map field not set",
		field:    "env",
		subfield: "ENV_VAR",
		err:      "cannot override.* field.*, not set",
	}, {
		desc:  "list missing subfield",
		field: "ports",
		err:   "cannot override.* without sub-field",
	}, {
		desc:     "list bad index",
		field:    "ports",
		subfield: "spam",
		err:      ".* sub-field must be an integer index",
	}, {
		desc:     "list index out of range",
		field:    "ports",
		subfield: "1",
		err:      ".* index 1 out of range",
	}}

	proc := &charm.Process{
		Name: "a proc",
		Type: "docker",
	}

	for i, t := range tests {
		t.log(c, i)
		var additions []charm.ProcessFieldValue
		overrides := t.changes()
		_, err := proc.Apply(overrides, additions)
		c.Assert(err, gc.NotNil)

		c.Check(err, gc.ErrorMatches, t.err)
	}
}

func (s *MetaSuite) TestProcessApplyBadAddition(c *gc.C) {
	tests := []procTest{{
		desc:  "unknown field",
		field: "spam",
		err:   "unrecognized field.*",
	}, {
		desc:  "name",
		field: "name",
		err:   ".* already set",
	}, {
		desc:  "type",
		field: "type",
		err:   ".* already set",
	}, {
		desc:  "simple field already set",
		field: "description",
		err:   ".* already set",
	}, {
		desc:  "map missing subfield",
		field: "env",
		err:   "cannot extend.* without sub-field",
	}, {
		desc:     "map field already set",
		field:    "env",
		subfield: "ENV_VAR",
		err:      ".* field.* already set",
	}, {
		desc:     "list unexpected subfield",
		field:    "ports",
		subfield: "10",
		err:      "cannot extend.* with sub-field",
	}}

	proc := &charm.Process{
		Name:        "a proc",
		Type:        "docker",
		Description: "my proc",
		EnvVars: map[string]string{
			"ENV_VAR": "yes",
		},
		Ports: []charm.ProcessPort{{
			External: 80,
			Internal: 8080,
		}},
	}

	for i, t := range tests {
		t.log(c, i)
		var overrides []charm.ProcessFieldValue
		additions := t.changes()
		_, err := proc.Apply(overrides, additions)
		c.Assert(err, gc.NotNil)

		c.Check(err, gc.ErrorMatches, t.err)
	}
}

func (s *MetaSuite) TestProcessNameRequired(c *gc.C) {
	proc := charm.Process{}
	c.Assert(proc.Validate(), gc.ErrorMatches, "missing name")
}

func (s *MetaSuite) TestProcesses(c *gc.C) {
	// "type" is the only required attribute for storage.
	meta, err := charm.ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
processes:
  proc0:
    description: a process
    type: docker
    type-options:
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
    type: rkt
`))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(meta.Processes, gc.DeepEquals, map[string]charm.Process{
		"proc0": {
			Name:        "proc0",
			Description: "a process",
			Type:        "docker",
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
	//c.Assert(err, gc.ErrorMatches, "metadata: processes.badproc.type: name is required")
	c.Assert(err, gc.ErrorMatches, "metadata: processes.badproc: expected map, got nothing")
}

func (s *MetaSuite) TestProcessesTypeNameRequired(c *gc.C) {
	badProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  badproc:
    foo: bar
`)
	_, err := charm.ReadMeta(badProc)
	c.Assert(err, gc.ErrorMatches, "metadata: processes.badproc.type: expected string, got nothing")
}

func (s *MetaSuite) TestProcessesPortEndpointFound(c *gc.C) {
	storageProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  endpointproc:
    type: docker
    ports:
        - <website>:8080
        - 443:8081
provides:
  website:
    interface: http
`)
	meta, err := charm.ReadMeta(storageProc)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(meta.Processes["endpointproc"].Ports[0].External, gc.Equals, 0)
	c.Check(meta.Processes["endpointproc"].Ports[0].Internal, gc.Equals, 8080)
	c.Check(meta.Processes["endpointproc"].Ports[0].Endpoint, gc.Equals, "website")
	c.Check(meta.Processes["endpointproc"].Ports[1].External, gc.Equals, 443)
	c.Check(meta.Processes["endpointproc"].Ports[1].Internal, gc.Equals, 8081)
	c.Check(meta.Processes["endpointproc"].Ports[1].Endpoint, gc.Equals, "")
}

func (s *MetaSuite) TestProcessesPortEndpointNotFound(c *gc.C) {
	storageProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  endpointproc:
    type: docker
    ports:
        - <website>:8080
        - 443:8081
provides:
  mysql:
    interface: db
`)
	_, err := charm.ReadMeta(storageProc)

	c.Assert(err, gc.ErrorMatches, `.* specified endpoint "website" unknown for .*`)
}

func (s *MetaSuite) TestProcessesStorageFound(c *gc.C) {
	storageProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  storageproc:
    type: docker
    volumes:
      - <store0>:/var/www/html:ro
storage:
    store0:
      type: filesystem
      location: /var/lib/things
`)
	meta, err := charm.ReadMeta(storageProc)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(meta.Processes["storageproc"].Volumes[0].ExternalMount, gc.Equals, "/var/lib/things")
	c.Check(meta.Processes["storageproc"].Volumes[0].Name, gc.Equals, "store0")
}

func (s *MetaSuite) TestProcessesStorageNotFound(c *gc.C) {
	storageProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  badproc:
    type: docker
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

func (s *MetaSuite) TestProcessesStorageNotFilesystem(c *gc.C) {
	storageProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  badproc:
    type: docker
    volumes:
      - <store0>:/var/www/html:ro
storage:
    store0:
        type: block
`)
	_, err := charm.ReadMeta(storageProc)
	c.Assert(err, gc.ErrorMatches, "metadata: processes.badproc.volumes: linked storage \"store0\" must be filesystem for .*")
}

func (s *MetaSuite) TestProcessesStorageMissingLocation(c *gc.C) {
	storageProc := strings.NewReader(`
name: a
summary: b
description: c
processes:
  badproc:
    type: docker
    volumes:
      - <store0>:/var/www/html:ro
storage:
    store0:
        type: filesystem
`)
	_, err := charm.ReadMeta(storageProc)
	c.Assert(err, gc.ErrorMatches, "metadata: processes.badproc.volumes: linked storage \"store0\" missing location for .*")
}
