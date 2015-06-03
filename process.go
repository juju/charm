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
	// External is the port on the host.
	External int
	// Internal is the port on the process.
	Internal int
}

// ProcessVolume is storage volume information for a workload process.
type ProcessVolume struct {
	// ExternalMount is the path on the host.
	ExternalMount string
	// InternalMount is the path on the process.
	InternalMount string
	// Mode is the "ro" OR "rw"
	Mode string
	// Storage is the name the metadata entry, if any.
	Storage string

	// storage is the storage that matched the Storage field.
	storage *Storage
}

// Process is the static definition of a workload process in a charm.
type Process struct {
	// Name is the name of the process.
	Name string
	// Description is a brief description of the process.
	Description string
	// Type is the name of the process type.
	Type string
	// TypeOptions is a map of arguments for the process type.
	TypeOptions map[string]string
	// Command is use command executed used by the process, if any.
	Command string
	// Image is the image used by the process, if any.
	Image string
	// Ports is a list of ProcessPort.
	Ports []ProcessPort
	// Volumes is a list of ProcessVolume.
	Volumes []ProcessVolume
	// EnvVars is map of environment variables used by the process.
	EnvVars map[string]string
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
func (p Process) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("missing name")
	}
	if p.Type == "" {
		return fmt.Errorf("metadata: processes.%s.type: name is required", p.Name)
	}

	for _, volume := range p.Volumes {
		if volume.Storage != "" && volume.ExternalMount == "" {
			if volume.storage == nil {
				return fmt.Errorf("metadata: processes.%s.volumes: specified storage %q unknown for %v", p.Name, volume.Storage, volume)
			}
			if volume.storage.Type != StorageFilesystem {
				return fmt.Errorf("metadata: processes.%s.volumes: linked storage %q must be filesystem for %v", p.Name, volume.Storage, volume)
			}
			if volume.storage.Location == "" {
				return fmt.Errorf("metadata: processes.%s.volumes: linked storage %q missing location for %v", p.Name, volume.Storage, volume)
			}
		}
	}

	return nil
}

func parseProcesses(data interface{}, storage map[string]Storage) map[string]Process {
	if data == nil {
		return nil
	}
	result := make(map[string]Process)
	for name, procData := range data.(map[string]interface{}) {
		procMap := procData.(map[string]interface{})
		result[name] = parseProcess(name, procMap, storage)
	}
	return result
}

func parseProcess(name string, coerced map[string]interface{}, storage map[string]Storage) Process {
	proc := Process{
		Name: name,
	}

	if description, ok := coerced["description"]; ok {
		proc.Description = description.(string)
	}

	if typeMap, ok := coerced["type"]; ok {
		options := typeMap.(map[string]interface{})
		// proc.Type validation is handled by Validate()
		proc.Type, _ = options["name"].(string)
		delete(options, "name")

		if len(options) > 0 {
			proc.TypeOptions = make(map[string]string)
			for k, v := range options {
				proc.TypeOptions[k] = v.(string)
			}
		}
	}

	if command, ok := coerced["command"]; ok {
		proc.Command = command.(string)
	}

	if image, ok := coerced["image"]; ok {
		proc.Image = image.(string)
	}

	if portsList, ok := coerced["ports"]; ok {
		for _, port := range portsList.([]interface{}) {
			proc.Ports = append(proc.Ports, *port.(*ProcessPort))
		}
	}

	if volumeList, ok := coerced["volumes"]; ok {
		for _, volumeRaw := range volumeList.([]interface{}) {
			volume := *volumeRaw.(*ProcessVolume)
			if volume.Storage != "" {
				volume.ExternalMount = ""
				for sName, s := range storage {
					if volume.Storage == sName {
						copied := s
						volume.storage = &copied
						if s.Type == StorageFilesystem {
							volume.ExternalMount = s.Location
						}
						break
					}
				}
			}
			proc.Volumes = append(proc.Volumes, volume)
		}
	}

	if envMap, ok := coerced["env"]; ok {
		proc.EnvVars = make(map[string]string)
		for k, v := range envMap.(map[string]interface{}) {
			proc.EnvVars[k] = v.(string)
		}
	}

	return proc
}

func checkProcesses(procs map[string]Process) error {
	for _, proc := range procs {
		if err := proc.Validate(); err != nil {
			return err
		}
	}
	return nil
}

var processSchema = schema.FieldMap(
	schema.Fields{
		"description": schema.String(),
		"type":        schema.StringMap(processTypeOptionChecker{}),
		"command":     schema.String(),
		"image":       schema.String(),
		"ports":       schema.List(processPortsChecker{}),
		"volumes":     schema.List(processVolumeChecker{}),
		"env":         schema.StringMap(schema.String()),
	},
	schema.Defaults{
		"description": schema.Omit,
		"command":     schema.Omit,
		"image":       schema.Omit,
		"ports":       schema.Omit,
		"volumes":     schema.Omit,
		"env":         schema.Omit,
	},
)

type processTypeOptionChecker struct{}

func (c processTypeOptionChecker) Coerce(v interface{}, path []string) (interface{}, error) {
	return fmt.Sprintf("%v", v), nil
}

type processPortsChecker struct{}

func (c processPortsChecker) Coerce(v interface{}, path []string) (interface{}, error) {
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

type processVolumeChecker struct{}

func (c processVolumeChecker) Coerce(v interface{}, path []string) (interface{}, error) {
	if _, err := schema.String().Coerce(v, path); err != nil {
		return nil, err
	}
	item := v.(string)

	parts := strings.SplitN(item, ":", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s: invalid value %q", strings.Join(path[1:], ""), item)
	}

	volume := ProcessVolume{
		ExternalMount: parts[0],
		InternalMount: parts[1],
	}

	if len(parts) == 3 {
		mode := parts[2]
		if _, err := schema.OneOf(schema.Const("rw"), schema.Const("ro")).Coerce(mode, path); err != nil {
			return nil, err
		}
		volume.Mode = mode
	}

	if strings.HasPrefix(volume.ExternalMount, "<") && strings.HasSuffix(volume.ExternalMount, ">") {
		volume.Storage = volume.ExternalMount[1 : len(volume.ExternalMount)-1]
	}
	return &volume, nil
}
