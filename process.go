// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/juju/schema"
)

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

// Copy create a deep copy of the Process.
func (copied Process) Copy() Process {
	typeOptions := make(map[string]string)
	for k, v := range copied.TypeOptions {
		typeOptions[k] = v
	}
	copied.TypeOptions = typeOptions

	envVars := make(map[string]string)
	for k, v := range copied.EnvVars {
		envVars[k] = v
	}
	copied.EnvVars = envVars

	var ports []ProcessPort
	for _, port := range copied.Ports {
		ports = append(ports, port)
	}
	copied.Ports = ports

	var volumes []ProcessVolume
	for _, volume := range copied.Volumes {
		volumes = append(volumes, volume.Copy())
	}
	copied.Volumes = volumes

	return copied
}

// ProcessFieldValue describes a requested change to a Process.
type ProcessFieldValue struct {
	Field    string
	Subfield string
	Value    string
}

// Override updates the Process with the provided value. If the
// identified field is not already set then Override fails.
func (p *Process) Override(value ProcessFieldValue) error {
	switch value.Field {
	case "name":
		// TODO(ericsnow) Allow overriding the name (for multiple copies)?
		return fmt.Errorf(`cannot override "name"`)
	case "description":
		if p.Description == "" {
			return fmt.Errorf(`cannot override "description", not set`)
		}
		p.Description = value.Value
	case "type":
		return fmt.Errorf(`cannot override "type"`)
	case "type-options":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot override "type-options" without sub-field`)
		}
		if _, ok := p.TypeOptions[value.Subfield]; !ok {
			return fmt.Errorf(`cannot override "type-options" field %q, not set`, value.Subfield)
		}
		p.TypeOptions[value.Subfield] = value.Value
	case "command":
		if p.Command == "" {
			return fmt.Errorf(`cannot override "command", not set`)
		}
		p.Command = value.Value
	case "image":
		if p.Image == "" {
			return fmt.Errorf(`cannot override "image", not set`)
		}
		p.Image = value.Value
	case "ports":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot override "ports" without sub-field`)
		}
		index, err := strconv.Atoi(value.Subfield)
		if err != nil {
			return fmt.Errorf(`"ports" sub-field must be an integer index`)
		}
		if index < 0 || index >= len(p.Ports) {
			return fmt.Errorf(`"ports" index %d out of range`, index)
		}
		var port ProcessPort
		if err := port.Set(value.Value); err != nil {
			return err
		}
		p.Ports[index] = port
	case "volumes":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot override "volumes" without sub-field`)
		}
		index, err := strconv.Atoi(value.Subfield)
		if err != nil {
			return fmt.Errorf(`"ports" sub-field must be an integer index`)
		}
		if index < 0 || index >= len(p.Ports) {
			return fmt.Errorf(`"ports" index %d out of range`, index)
		}
		var volume ProcessVolume
		if err := volume.Set(value.Value); err != nil {
			return err
		}
		p.Volumes[index] = volume
	case "env":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot override "env" without sub-field`)
		}
		if _, ok := p.EnvVars[value.Subfield]; !ok {
			return fmt.Errorf(`cannot override "env" field %q, not set`, value.Subfield)
		}
		p.EnvVars[value.Subfield] = value.Value
	default:
		return fmt.Errorf("unrecognized field %q", value.Field)
	}
	return nil
}

// Extend updates the Process with the provided value. If the
// identified field is already set then Extend fails.
func (p *Process) Extend(value ProcessFieldValue) error {
	switch value.Field {
	case "name":
		// TODO(ericsnow) Allow overriding the name (for multiple copies)?
		return fmt.Errorf(`"name" already set`)
	case "description":
		if p.Description != "" {
			return fmt.Errorf(`"description" already set`)
		}
		p.Description = value.Value
	case "type":
		return fmt.Errorf(`"type" already set`)
	case "type-options":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot extend "type-options" without sub-field`)
		}
		if p.TypeOptions == nil {
			p.TypeOptions = make(map[string]string)
		} else if _, ok := p.TypeOptions[value.Subfield]; ok {
			return fmt.Errorf(`"type-options" field %q already set`, value.Subfield)
		}
		p.TypeOptions[value.Subfield] = value.Value
	case "command":
		if p.Command != "" {
			return fmt.Errorf(`cannot extend "command" already set`)
		}
		p.Command = value.Value
	case "image":
		if p.Image != "" {
			return fmt.Errorf(`cannot extend "image" already set`)
		}
		p.Image = value.Value
	case "ports":
		if value.Subfield != "" {
			return fmt.Errorf(`cannot extend "ports" with sub-field`)
		}
		var port ProcessPort
		if err := port.Set(value.Value); err != nil {
			return err
		}
		p.Ports = append(p.Ports, port)
	case "volumes":
		if value.Subfield != "" {
			return fmt.Errorf(`cannot extend "volumes" with sub-field`)
		}
		var volume ProcessVolume
		if err := volume.Set(value.Value); err != nil {
			return err
		}
		p.Volumes = append(p.Volumes, volume)
	case "env":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot extend "env" without sub-field`)
		}
		if p.EnvVars == nil {
			p.EnvVars = make(map[string]string)
		} else if _, ok := p.EnvVars[value.Subfield]; ok {
			return fmt.Errorf(`"env" field %q already set`, value.Subfield)
		}
		p.EnvVars[value.Subfield] = value.Value
	default:
		return fmt.Errorf("unrecognized field %q", value.Field)
	}
	return nil
}

// Apply makes a copy of the Process and applies the given overrides
// and additions to that copy.
func (p *Process) Apply(overrides []ProcessFieldValue, additions []ProcessFieldValue) (*Process, error) {
	process := p.Copy()
	for _, value := range overrides {
		if err := process.Override(value); err != nil {
			return nil, err
		}
	}
	for _, value := range additions {
		if err := process.Extend(value); err != nil {
			return nil, err
		}
	}
	return &process, nil
}

// Validate checks the Process for errors.
func (p Process) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("missing name")
	}
	if p.Type == "" {
		return fmt.Errorf("metadata: processes.%s.type: name is required", p.Name)
	}

	if err := p.validatePorts(); err != nil {
		return err
	}

	if err := p.validateStorage(); err != nil {
		return err
	}

	return nil
}

func (p Process) validatePorts() error {
	for _, port := range p.Ports {
		if port.External < 0 {
			return fmt.Errorf("metadata: processes.%s.ports: specified endpoint %q unknown for %v", p.Name, port.Endpoint, port)
		}
	}
	return nil
}

func (p Process) validateStorage() error {
	for _, volume := range p.Volumes {
		if volume.Name != "" && volume.ExternalMount == "" {
			if volume.storage == nil {
				return fmt.Errorf("metadata: processes.%s.volumes: specified storage %q unknown for %v", p.Name, volume.Name, volume)
			}
			if volume.storage.Type != StorageFilesystem {
				return fmt.Errorf("metadata: processes.%s.volumes: linked storage %q must be filesystem for %v", p.Name, volume.Name, volume)
			}
			if volume.storage.Location == "" {
				return fmt.Errorf("metadata: processes.%s.volumes: linked storage %q missing location for %v", p.Name, volume.Name, volume)
			}
		}
	}
	return nil
}

// ProcessPort is network port information for a workload process.
type ProcessPort struct {
	// External is the port on the host.
	External int
	// Internal is the port on the process.
	Internal int
	// Endpoint is the unit-relation endpoint matching the external
	// port, if any.
	Endpoint string
}

// Set parses the provided string and sets the appropriate fields.
func (p *ProcessPort) Set(raw string) error {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid value %q", raw)
	}
	if err := p.SetExternal(parts[0]); err != nil {
		return err
	}
	if err := p.SetInternal(parts[1]); err != nil {
		return err
	}
	return nil
}

// SetExternal parses the provided string and sets the appropriate fields.
func (p *ProcessPort) SetExternal(portStr string) error {
	p.External = 0
	p.Endpoint = ""
	if strings.HasPrefix(portStr, "<") && strings.HasSuffix(portStr, ">") {
		// The port was specified by a relation endpoint rather than a
		// port number.
		p.Endpoint = portStr[1 : len(portStr)-1]
	} else {
		// It's just a port number.
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("expected int got %q", portStr)
		}
		p.External = port
	}
	return nil
}

// SetInternal parses the provided string and sets the appropriate fields.
func (p *ProcessPort) SetInternal(portStr string) error {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("expected int got %q", portStr)
	}
	p.Internal = port
	return nil
}

// ProcessVolume is storage volume information for a workload process.
type ProcessVolume struct {
	// ExternalMount is the path on the host.
	ExternalMount string
	// InternalMount is the path on the process.
	InternalMount string
	// Mode is the "ro" OR "rw"
	Mode string
	// Name is the name of the storage metadata entry, if any.
	Name string

	// storage is the storage that matched the Storage field.
	storage *Storage
}

// Copy create a deep copy of the ProcessVolume.
func (copied ProcessVolume) Copy() ProcessVolume {
	copied.storage = nil
	return copied
}

// Set parses the provided string and sets the appropriate fields.
func (pv *ProcessVolume) Set(raw string) error {
	parts := strings.SplitN(raw, ":", 3)
	if len(parts) < 2 {
		return fmt.Errorf("invalid value %q", raw)
	}
	pv.SetExternal(parts[0])
	pv.SetInternal(parts[1])
	if len(parts) == 3 {
		if err := pv.SetMode(parts[2]); err != nil {
			return err
		}
	}
	return nil
}

// SetExternal parses the provided string and sets the appropriate fields.
func (pv *ProcessVolume) SetExternal(volume string) {
	pv.Name = ""
	pv.ExternalMount = ""
	if strings.HasPrefix(volume, "<") && strings.HasSuffix(volume, ">") {
		// It's a reference to a defined storage attachment.
		pv.Name = volume[1 : len(volume)-1]
	} else {
		// It's just a volume name.
		pv.ExternalMount = volume
	}
}

// SetInternal parses the provided string and sets the appropriate fields.
func (pv *ProcessVolume) SetInternal(volume string) {
	pv.InternalMount = volume
}

// SetMode parses the provided string and sets the appropriate fields.
func (pv *ProcessVolume) SetMode(mode string) error {
	if _, err := schema.OneOf(schema.Const("rw"), schema.Const("ro")).Coerce(mode, nil); err != nil {
		return fmt.Errorf(`expected "rw" or "ro" for mode, got %q`, mode)
	}
	pv.Mode = mode
	return nil
}

func parseProcesses(data interface{}, provides map[string]Relation, storage map[string]Storage) map[string]Process {
	if data == nil {
		return nil
	}
	result := make(map[string]Process)
	for name, procData := range data.(map[string]interface{}) {
		procMap := procData.(map[string]interface{})
		result[name] = parseProcess(name, procMap, provides, storage)
	}
	return result
}

func parseProcess(name string, coerced map[string]interface{}, provides map[string]Relation, storage map[string]Storage) Process {
	proc := Process{
		Name: name,
		Type: coerced["type"].(string),
	}

	if description, ok := coerced["description"]; ok {
		proc.Description = description.(string)
	}

	if typeMap, ok := coerced["type-options"]; ok {
		options := typeMap.(map[string]interface{})
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
		for _, portRaw := range portsList.([]interface{}) {
			port := portRaw.(*ProcessPort)
			if port.External == 0 {
				port.External = -1
				for endpoint := range provides {
					if port.Endpoint == endpoint {
						port.External = 0
						break
					}
				}
			}
			proc.Ports = append(proc.Ports, *port)
		}
	}

	if volumeList, ok := coerced["volumes"]; ok {
		for _, volumeRaw := range volumeList.([]interface{}) {
			volume := *volumeRaw.(*ProcessVolume)
			if volume.Name != "" {
				volume.ExternalMount = ""
				for sName, s := range storage {
					if volume.Name == sName {
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
		"description":  schema.String(),
		"type":         schema.String(),
		"type-options": schema.StringMap(forcedStringChecker{}),
		"command":      schema.String(),
		"image":        schema.String(),
		"ports":        schema.List(processPortsChecker{}),
		"volumes":      schema.List(processVolumeChecker{}),
		"env":          schema.StringMap(forcedStringChecker{}),
	},
	schema.Defaults{
		"description":  schema.Omit,
		"type-options": schema.Omit,
		"command":      schema.Omit,
		"image":        schema.Omit,
		"ports":        schema.Omit,
		"volumes":      schema.Omit,
		"env":          schema.Omit,
	},
)

type forcedStringChecker struct{}

// Coerce implements schema.Checker.
func (c forcedStringChecker) Coerce(v interface{}, path []string) (interface{}, error) {
	_, err := schema.OneOf(
		schema.Bool(),
		schema.Int(),
		schema.Float(),
		schema.String(),
	).Coerce(v, path)
	if err != nil {
		return nil, err
	}
	return fmt.Sprint(v), nil
}

type processPortsChecker struct{}

// Coerce implements schema.Checker.
func (c processPortsChecker) Coerce(v interface{}, path []string) (interface{}, error) {
	if _, err := schema.String().Coerce(v, path); err != nil {
		return nil, err
	}
	item := v.(string)

	var port ProcessPort
	if err := port.Set(item); err != nil {
		return nil, fmt.Errorf("%s: %v", strings.Join(path[1:], ""), err)
	}
	return &port, nil
}

type processVolumeChecker struct{}

// Coerce implements schema.Checker.
func (c processVolumeChecker) Coerce(v interface{}, path []string) (interface{}, error) {
	if _, err := schema.String().Coerce(v, path); err != nil {
		return nil, err
	}
	item := v.(string)

	var volume ProcessVolume
	if err := volume.Set(item); err != nil {
		return nil, fmt.Errorf("%s: %v", strings.Join(path[1:], ""), err)
	}
	return &volume, nil
}
