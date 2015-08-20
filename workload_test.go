package charm_test

import (
	"strings"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/yaml.v1"

	"gopkg.in/juju/charm.v5"
)

func (s *MetaSuite) TestWorkloadParseOkay(c *gc.C) {
	raw := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(`
description: a workload
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
	workload, err := charm.ParseWorkload("workload0", raw)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(workload, jc.DeepEquals, &charm.Workload{
		Name:        "workload0",
		Description: "a workload",
		Type:        "docker",
		TypeOptions: map[string]string{
			"publish_all": "true",
		},
		Command: "foocmd",
		Image:   "nginx/nginx",
		Ports: []charm.WorkloadPort{{
			External: 80,
			Internal: 8080,
		}, {
			External: 443,
			Internal: 8081,
		}},
		Volumes: []charm.WorkloadVolume{{
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

func (s *MetaSuite) TestWorkloadParseMinimal(c *gc.C) {
	raw := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(`
type: docker
`), raw)
	c.Assert(err, jc.ErrorIsNil)
	workload, err := charm.ParseWorkload("workload0", raw)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(workload, jc.DeepEquals, &charm.Workload{
		Name:        "workload0",
		Description: "",
		Type:        "docker",
		TypeOptions: nil,
		Command:     "",
		Image:       "",
		Ports:       nil,
		Volumes:     nil,
		EnvVars:     nil,
	})
	c.Check(workload, jc.DeepEquals, &charm.Workload{
		Name: "workload0",
		Type: "docker",
	})
}

func (s *MetaSuite) TestWorkloadCopyVolume(c *gc.C) {
	vol := charm.WorkloadVolume{
		ExternalMount: "a",
		InternalMount: "b",
		Mode:          "ro",
		Name:          "spam",
	}
	copied := vol.Copy()

	c.Check(copied, jc.DeepEquals, vol)
}

func (s *MetaSuite) TestWorkloadCopyWorkloadOkay(c *gc.C) {
	workload := charm.Workload{
		Name:        "workload0",
		Description: "a workload",
		Type:        "docker",
		TypeOptions: map[string]string{
			"publish_all": "true",
		},
		Command: "foocmd",
		Image:   "nginx/nginx",
		Ports: []charm.WorkloadPort{{
			External: 80,
			Internal: 8080,
		}, {
			External: 443,
			Internal: 8081,
		}},
		Volumes: []charm.WorkloadVolume{{
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
	copied := workload.Copy()

	c.Check(copied, jc.DeepEquals, workload)
}

func (s *MetaSuite) TestWorkloadCopyWorkloadMinimal(c *gc.C) {
	workload := charm.Workload{
		Name: "workload0",
		Type: "docker",
	}
	copied := workload.Copy()

	c.Check(copied, jc.DeepEquals, workload)
}

func (s *MetaSuite) TestWorkloadApplyOkay(c *gc.C) {
	workload := &charm.Workload{
		Name: "a workload",
		Type: "docker",
		TypeOptions: map[string]string{
			"publish_all": "true",
		},
		Image: "nginx/nginx-2",
		Ports: []charm.WorkloadPort{{
			External: 81,
			Internal: 8001,
		}},
		Volumes: []charm.WorkloadVolume{{
			ExternalMount: "/var/www/html",
			InternalMount: "/usr/share/nginx/html",
			Mode:          "rw",
		}},
		EnvVars: map[string]string{
			"ENV_VAR": "spam",
		},
	}
	overrides := []charm.WorkloadFieldValue{{
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
	additions := []charm.WorkloadFieldValue{{
		Field: "description",
		Value: "my workload",
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
	applied, err := workload.Apply(overrides, additions)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(applied, jc.DeepEquals, &charm.Workload{
		Name:        "a workload",
		Type:        "docker",
		Description: "my workload",
		TypeOptions: map[string]string{
			"publish_all": "NO",
		},
		Command: "foocmd",
		Image:   "nginx/nginx",
		Ports: []charm.WorkloadPort{{
			External: 80,
			Internal: 8080,
		}, {
			External: 443,
			Internal: 8081,
		}},
		Volumes: []charm.WorkloadVolume{{
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

func (s *MetaSuite) TestWorkloadApplyEmpty(c *gc.C) {
	workload := &charm.Workload{}
	var overrides []charm.WorkloadFieldValue
	additions := []charm.WorkloadFieldValue{{
		Field:    "type-options",
		Subfield: "publish_all",
		Value:    "NO",
	}, {
		Field: "description",
		Value: "my workload",
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
	applied, err := workload.Apply(overrides, additions)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(applied, jc.DeepEquals, &charm.Workload{
		Description: "my workload",
		TypeOptions: map[string]string{
			"publish_all": "NO",
		},
		Command: "foocmd",
		Image:   "nginx/nginx",
		Ports: []charm.WorkloadPort{{
			External: 80,
			Internal: 8080,
		}, {
			External: 443,
			Internal: 8081,
		}},
		Volumes: []charm.WorkloadVolume{{
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

func (s *MetaSuite) TestWorkloadApplyMinimal(c *gc.C) {
	workload := &charm.Workload{
		Name:  "workload0",
		Type:  "docker",
		Image: "nginx/nginx",
	}
	overrides := []charm.WorkloadFieldValue{{
		Field: "image",
		Value: "nginx/nginx-2",
	}}
	additions := []charm.WorkloadFieldValue{{
		Field: "description",
		Value: "my workload",
	}}
	applied, err := workload.Apply(overrides, additions)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(applied, jc.DeepEquals, &charm.Workload{
		Name:        "workload0",
		Description: "my workload",
		Type:        "docker",
		Image:       "nginx/nginx-2",
	})
}

func (s *MetaSuite) TestWorkloadApplyNoChange(c *gc.C) {
	workload := &charm.Workload{
		Name:        "a workload",
		Type:        "docker",
		Description: "my workload",
		TypeOptions: map[string]string{
			"publish_all": "NO",
		},
		Command: "foocmd",
		Image:   "nginx/nginx",
		Ports: []charm.WorkloadPort{{
			External: 80,
			Internal: 8080,
		}, {
			External: 443,
			Internal: 8081,
		}},
		Volumes: []charm.WorkloadVolume{{
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
	var overrides, additions []charm.WorkloadFieldValue
	applied, err := workload.Apply(overrides, additions)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(applied, jc.DeepEquals, &charm.Workload{
		Name:        "a workload",
		Type:        "docker",
		Description: "my workload",
		TypeOptions: map[string]string{
			"publish_all": "NO",
		},
		Command: "foocmd",
		Image:   "nginx/nginx",
		Ports: []charm.WorkloadPort{{
			External: 80,
			Internal: 8080,
		}, {
			External: 443,
			Internal: 8081,
		}},
		Volumes: []charm.WorkloadVolume{{
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

type workloadTest struct {
	desc     string
	field    string
	subfield string
	value    string
	err      string
}

func (t workloadTest) log(c *gc.C, i int) {
	c.Logf("test %d: %s", i, t.desc)
}

func (t workloadTest) changes() []charm.WorkloadFieldValue {
	return []charm.WorkloadFieldValue{{
		Field:    t.field,
		Subfield: t.subfield,
		Value:    t.value,
	}}
}

func (s *MetaSuite) TestWorkloadApplyBadOverride(c *gc.C) {
	tests := []workloadTest{{
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

	workload := &charm.Workload{
		Name: "a workload",
		Type: "docker",
	}

	for i, t := range tests {
		t.log(c, i)
		var additions []charm.WorkloadFieldValue
		overrides := t.changes()
		_, err := workload.Apply(overrides, additions)
		c.Assert(err, gc.NotNil)

		c.Check(err, gc.ErrorMatches, t.err)
	}
}

func (s *MetaSuite) TestWorkloadApplyBadAddition(c *gc.C) {
	tests := []workloadTest{{
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

	workload := &charm.Workload{
		Name:        "a workload",
		Type:        "docker",
		Description: "my workload",
		EnvVars: map[string]string{
			"ENV_VAR": "yes",
		},
		Ports: []charm.WorkloadPort{{
			External: 80,
			Internal: 8080,
		}},
	}

	for i, t := range tests {
		t.log(c, i)
		var overrides []charm.WorkloadFieldValue
		additions := t.changes()
		_, err := workload.Apply(overrides, additions)
		c.Assert(err, gc.NotNil)

		c.Check(err, gc.ErrorMatches, t.err)
	}
}

func (s *MetaSuite) TestWorkloadNameRequired(c *gc.C) {
	workload := charm.Workload{}
	c.Assert(workload.Validate(), gc.ErrorMatches, "missing name")
}

func (s *MetaSuite) TestWorkloads(c *gc.C) {
	// "type" is the only required attribute for storage.
	meta, err := charm.ReadMeta(strings.NewReader(`
name: a
summary: b
description: c
workloads:
  workload0:
    description: a workload
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
  workload1:
    type: rkt
`))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(meta.Workloads, gc.DeepEquals, map[string]charm.Workload{
		"workload0": {
			Name:        "workload0",
			Description: "a workload",
			Type:        "docker",
			TypeOptions: map[string]string{
				"publish_all": "true",
			},
			Command: "foocmd",
			Image:   "nginx/nginx",
			Ports: []charm.WorkloadPort{{
				External: 80,
				Internal: 8080,
			}, {
				External: 443,
				Internal: 8081,
			}},
			Volumes: []charm.WorkloadVolume{{
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
		"workload1": {
			Name: "workload1",
			Type: "rkt",
		},
	})
}

func (s *MetaSuite) TestWorkloadsNotRequired(c *gc.C) {
	noWorkload := strings.NewReader(`
name: a
summary: b
description: c
`)
	_, err := charm.ReadMeta(noWorkload)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *MetaSuite) TestWorkloadsTypeRequired(c *gc.C) {
	badWorkload := strings.NewReader(`
name: a
summary: b
description: c
workloads:
  badworkload:
`)
	_, err := charm.ReadMeta(badWorkload)
	//c.Assert(err, gc.ErrorMatches, "metadata: workloads.badworkload.type: name is required")
	c.Assert(err, gc.ErrorMatches, "metadata: workloads.badworkload: expected map, got nothing")
}

func (s *MetaSuite) TestWorkloadsTypeNameRequired(c *gc.C) {
	badWorkload := strings.NewReader(`
name: a
summary: b
description: c
workloads:
  badworkload:
    foo: bar
`)
	_, err := charm.ReadMeta(badWorkload)
	c.Assert(err, gc.ErrorMatches, "metadata: workloads.badworkload.type: expected string, got nothing")
}

func (s *MetaSuite) TestWorkloadsPortEndpointFound(c *gc.C) {
	storageWorkload := strings.NewReader(`
name: a
summary: b
description: c
workloads:
  endpointworkload:
    type: docker
    ports:
        - <website>:8080
        - 443:8081
provides:
  website:
    interface: http
`)
	meta, err := charm.ReadMeta(storageWorkload)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(meta.Workloads["endpointworkload"].Ports[0].External, gc.Equals, 0)
	c.Check(meta.Workloads["endpointworkload"].Ports[0].Internal, gc.Equals, 8080)
	c.Check(meta.Workloads["endpointworkload"].Ports[0].Endpoint, gc.Equals, "website")
	c.Check(meta.Workloads["endpointworkload"].Ports[1].External, gc.Equals, 443)
	c.Check(meta.Workloads["endpointworkload"].Ports[1].Internal, gc.Equals, 8081)
	c.Check(meta.Workloads["endpointworkload"].Ports[1].Endpoint, gc.Equals, "")
}

func (s *MetaSuite) TestWorkloadsPortEndpointNotFound(c *gc.C) {
	storageWorkload := strings.NewReader(`
name: a
summary: b
description: c
workloads:
  endpointworkload:
    type: docker
    ports:
        - <website>:8080
        - 443:8081
provides:
  mysql:
    interface: db
`)
	_, err := charm.ReadMeta(storageWorkload)

	c.Assert(err, gc.ErrorMatches, `.* specified endpoint "website" unknown for .*`)
}

func (s *MetaSuite) TestWorkloadsStorageFound(c *gc.C) {
	storageWorkload := strings.NewReader(`
name: a
summary: b
description: c
workloads:
  storageworkload:
    type: docker
    volumes:
      - <store0>:/var/www/html:ro
storage:
    store0:
      type: filesystem
      location: /var/lib/things
`)
	meta, err := charm.ReadMeta(storageWorkload)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(meta.Workloads["storageworkload"].Volumes[0].ExternalMount, gc.Equals, "/var/lib/things")
	c.Check(meta.Workloads["storageworkload"].Volumes[0].Name, gc.Equals, "store0")
}

func (s *MetaSuite) TestWorkloadsStorageNotFound(c *gc.C) {
	storageWorkload := strings.NewReader(`
name: a
summary: b
description: c
workloads:
  badworkload:
    type: docker
    volumes:
      - <store1>:/var/www/html:ro
storage:
    store0:
        type: filesystem
        location: /var/lib/things
`)
	_, err := charm.ReadMeta(storageWorkload)
	c.Assert(err, gc.ErrorMatches, "metadata: workloads.badworkload.volumes: specified storage \"store1\" unknown for .*")
}

func (s *MetaSuite) TestWorkloadsStorageNotFilesystem(c *gc.C) {
	storageWorkload := strings.NewReader(`
name: a
summary: b
description: c
workloads:
  badworkload:
    type: docker
    volumes:
      - <store0>:/var/www/html:ro
storage:
    store0:
        type: block
`)
	_, err := charm.ReadMeta(storageWorkload)
	c.Assert(err, gc.ErrorMatches, "metadata: workloads.badworkload.volumes: linked storage \"store0\" must be filesystem for .*")
}

func (s *MetaSuite) TestWorkloadsStorageMissingLocation(c *gc.C) {
	storageWorkload := strings.NewReader(`
name: a
summary: b
description: c
workloads:
  badworkload:
    type: docker
    volumes:
      - <store0>:/var/www/html:ro
storage:
    store0:
        type: filesystem
`)
	_, err := charm.ReadMeta(storageWorkload)
	c.Assert(err, gc.ErrorMatches, "metadata: workloads.badworkload.volumes: linked storage \"store0\" missing location for .*")
}
