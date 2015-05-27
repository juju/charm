// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/juju/schema"
)

// ProcessPort is network port information for a workload process.
type ProcessPort struct {
	PortA int
	PortB int
}

// ProcessVolume is storage volume information for a workload process.
type ProcessVolume struct {
	ConcreteMount string
	VirtualMount  string
	Mode          string
	Storage       string
}

// Process is the static definition of a workload process in a charm.
type Process struct {
	Name        string
	Type        string
	TypeOptions map[string]string
	Command     string
	Image       string
	Ports       []ProcessPort
	Volumes     []ProcessVolume
	EnvVars     map[string]string
}

// ParseProcess parses the provided data and converts it to a Process.
// The data will most likely have been de-serialized, perhaps from YAML.
func ParseProcess(name string, data map[string]interface{}) (*Process, error) {
	raw, err := processSchema.Coerce(data, []string{name})
	if err != nil {
		return nil, err
	}
	return raw.(*Process), nil
}

// Validate checks the Process for errors.
func (p Process) Validate(storage map[string]Storage) error {
	if p.Name == "" {
		return fmt.Errorf("missing name")
	}
	if p.Type == "" {
		return fmt.Errorf("missing type")
	}

	for _, volume := range p.Volumes {
		if volume.Storage != "" {
			var matched bool
			for name := range storage {
				if volume.Storage == name {
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf("specified storage %q unknown for %v", volume.Storage, volume)
			}
		}
	}

	return nil
}

func parseProcesses(data interface{}) map[string]Process {
	if data == nil {
		return nil
	}
	result := make(map[string]Process)
	for name, procData := range data.(map[string]interface{}) {
		procMap := procData.(map[string]interface{})
		result[name] = parseProcess(name, procMap)
	}
	return result
}

func parseProcess(name string, coerced map[string]interface{}) Process {
	proc := Process{
		Name: name,
	}

	if typeMap, ok := coerced["type"]; ok {
		options := typeMap.(map[string]string)
		proc.Type, _ = options["name"]
		delete(options, "name")
		proc.TypeOptions = options
	}

	if command, ok := coerced["command"]; ok {
		proc.Command = command.(string)
	}

	if image, ok := coerced["image"]; ok {
		proc.Image = image.(string)
	}

	if portsList, ok := coerced["ports"]; ok {
		proc.Ports = portsList.([]ProcessPort)
	}

	if volumeList, ok := coerced["volumes"]; ok {
		proc.Volumes = volumeList.([]ProcessVolume)
	}

	if envMap, ok := coerced["env"]; ok {
		proc.EnvVars = envMap.(map[string]string)
	}

	return proc
}

func checkProcesses(procs map[string]Process, storage map[string]Storage) error {
	for _, proc := range procs {
		if err := proc.Validate(storage); err != nil {
			return err
		}
	}
	return nil
}

var processSchema = schema.FieldMap(
	schema.Fields{
		"type":    schema.StringMap(schema.String()),
		"command": schema.String(),
		"image":   schema.String(),
		"ports":   schema.List(processPortsSchema{}),
		"volumes": schema.List(processVolumeSchema{}),
		"env":     schema.StringMap(schema.String()),
	},
	schema.Defaults{
		"command": schema.Omit,
		"image":   schema.Omit,
		"ports":   schema.Omit,
		"volumes": schema.Omit,
		"env":     schema.Omit,
	},
)

type processPortsSchema struct{}

func (c processPortsSchema) Coerce(v interface{}, path []string) (interface{}, error) {
	if _, err := schema.String().Coerce(v, path); err != nil {
		return nil, err
	}
	item := v.(string)

	parts := strings.SplitN(item, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%s: invalid value %q", strings.Join(path[1:], ""), item)
	}

	portA, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%s: expected int got %q", strings.Join(path[1:], ""), parts[0])
	}

	portB, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%s: expected int got %q", strings.Join(path[1:], ""), parts[1])
	}

	return &ProcessPort{portA, portB}, nil
}

type processVolumeSchema struct{}

func (c processVolumeSchema) Coerce(v interface{}, path []string) (interface{}, error) {
	if _, err := schema.String().Coerce(v, path); err != nil {
		return nil, err
	}
	item := v.(string)

	parts := strings.SplitN(item, ":", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s: invalid value %q", strings.Join(path[1:], ""), item)
	}

	volume := ProcessVolume{
		ConcreteMount: parts[0],
		VirtualMount:  parts[1],
	}

	if len(parts) == 3 {
		mode := parts[2]
		if _, err := schema.OneOf(schema.Const("rw"), schema.Const("ro")).Coerce(mode, path); err != nil {
			return nil, err
		}
		volume.Mode = mode
	}

	if strings.HasPrefix(volume.ConcreteMount, "{") && strings.HasSuffix(volume.ConcreteMount, "}") {
		volume.Storage = volume.ConcreteMount[1 : len(volume.ConcreteMount)-1]
	}
	return &volume, nil
}
