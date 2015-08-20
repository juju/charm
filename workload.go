// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/juju/schema"
)

// Workload is the static definition of a workload workload in a charm.
type Workload struct {
	// Name is the name of the workload.
	Name string
	// Description is a brief description of the workload.
	Description string
	// Type is the name of the workload type.
	Type string
	// TypeOptions is a map of arguments for the workload type.
	TypeOptions map[string]string
	// Command is use command executed used by the workload, if any.
	Command string
	// Image is the image used by the workload, if any.
	Image string
	// Ports is a list of WorkloadPort.
	Ports []WorkloadPort
	// Volumes is a list of WorkloadVolume.
	Volumes []WorkloadVolume
	// EnvVars is map of environment variables used by the workload.
	EnvVars map[string]string
}

// ParseWorkload parses the provided data and converts it to a Workload.
// The data will most likely have been de-serialized, perhaps from YAML.
func ParseWorkload(name string, data map[interface{}]interface{}) (*Workload, error) {
	return ParseWorkloadWithRefs(name, data, nil, nil)
}

// ParseWorkloadWithRefs parses the provided data and converts it to a
// Workload. The data will most likely have been de-serialized, perhaps
// from YAML.
func ParseWorkloadWithRefs(name string, data map[interface{}]interface{}, provides map[string]Relation, storage map[string]Storage) (*Workload, error) {
	raw, err := workloadSchema.Coerce(data, []string{name})
	if err != nil {
		return nil, err
	}
	workload := parseWorkload(name, raw.(map[string]interface{}), provides, storage)
	if err := workload.Validate(); err != nil {
		return nil, err
	}
	return &workload, nil
}

// Copy create a deep copy of the Workload.
func (copied Workload) Copy() Workload {
	if copied.TypeOptions != nil {
		typeOptions := make(map[string]string)
		for k, v := range copied.TypeOptions {
			typeOptions[k] = v
		}
		copied.TypeOptions = typeOptions
	}

	if copied.EnvVars != nil {
		envVars := make(map[string]string)
		for k, v := range copied.EnvVars {
			envVars[k] = v
		}
		copied.EnvVars = envVars
	}

	var ports []WorkloadPort
	for _, port := range copied.Ports {
		ports = append(ports, port)
	}
	copied.Ports = ports

	var volumes []WorkloadVolume
	for _, volume := range copied.Volumes {
		volumes = append(volumes, volume.Copy())
	}
	copied.Volumes = volumes

	return copied
}

// WorkloadFieldValue describes a requested change to a Workload.
type WorkloadFieldValue struct {
	// Field is the name of the metadata field.
	Field string
	// Field is the name of the metadata sub-field, if applicable.
	Subfield string
	// Value is the value to assign to the field.
	Value string
}

// Override updates the Workload with the provided value. If the
// identified field is not already set then Override fails.
func (w *Workload) Override(value WorkloadFieldValue) error {
	switch value.Field {
	case "name":
		// TODO(ericsnow) Allow overriding the name (for multiple copies)?
		return fmt.Errorf(`cannot override "name"`)
	case "description":
		if w.Description == "" {
			return fmt.Errorf(`cannot override "description", not set`)
		}
		w.Description = value.Value
	case "type":
		return fmt.Errorf(`cannot override "type"`)
	case "type-options":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot override "type-options" without sub-field`)
		}
		if _, ok := w.TypeOptions[value.Subfield]; !ok {
			return fmt.Errorf(`cannot override "type-options" field %q, not set`, value.Subfield)
		}
		w.TypeOptions[value.Subfield] = value.Value
	case "command":
		if w.Command == "" {
			return fmt.Errorf(`cannot override "command", not set`)
		}
		w.Command = value.Value
	case "image":
		if w.Image == "" {
			return fmt.Errorf(`cannot override "image", not set`)
		}
		w.Image = value.Value
	case "ports":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot override "ports" without sub-field`)
		}
		index, err := strconv.Atoi(value.Subfield)
		if err != nil {
			return fmt.Errorf(`"ports" sub-field must be an integer index`)
		}
		if index < 0 || index >= len(w.Ports) {
			return fmt.Errorf(`"ports" index %d out of range`, index)
		}
		var port WorkloadPort
		if err := port.Set(value.Value); err != nil {
			return err
		}
		w.Ports[index] = port
	case "volumes":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot override "volumes" without sub-field`)
		}
		index, err := strconv.Atoi(value.Subfield)
		if err != nil {
			return fmt.Errorf(`"ports" sub-field must be an integer index`)
		}
		if index < 0 || index >= len(w.Ports) {
			return fmt.Errorf(`"ports" index %d out of range`, index)
		}
		var volume WorkloadVolume
		if err := volume.Set(value.Value); err != nil {
			return err
		}
		w.Volumes[index] = volume
	case "env":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot override "env" without sub-field`)
		}
		if _, ok := w.EnvVars[value.Subfield]; !ok {
			return fmt.Errorf(`cannot override "env" field %q, not set`, value.Subfield)
		}
		w.EnvVars[value.Subfield] = value.Value
	default:
		return fmt.Errorf("unrecognized field %q", value.Field)
	}
	return nil
}

// Extend updates the Workload with the provided value. If the
// identified field is already set then Extend fails.
func (w *Workload) Extend(value WorkloadFieldValue) error {
	switch value.Field {
	case "name":
		// TODO(ericsnow) Allow overriding the name (for multiple copies)?
		return fmt.Errorf(`"name" already set`)
	case "description":
		if w.Description != "" {
			return fmt.Errorf(`"description" already set`)
		}
		w.Description = value.Value
	case "type":
		return fmt.Errorf(`"type" already set`)
	case "type-options":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot extend "type-options" without sub-field`)
		}
		if w.TypeOptions == nil {
			w.TypeOptions = make(map[string]string)
		} else if _, ok := w.TypeOptions[value.Subfield]; ok {
			return fmt.Errorf(`"type-options" field %q already set`, value.Subfield)
		}
		w.TypeOptions[value.Subfield] = value.Value
	case "command":
		if w.Command != "" {
			return fmt.Errorf(`cannot extend "command" already set`)
		}
		w.Command = value.Value
	case "image":
		if w.Image != "" {
			return fmt.Errorf(`cannot extend "image" already set`)
		}
		w.Image = value.Value
	case "ports":
		if value.Subfield != "" {
			return fmt.Errorf(`cannot extend "ports" with sub-field`)
		}
		var port WorkloadPort
		if err := port.Set(value.Value); err != nil {
			return err
		}
		w.Ports = append(w.Ports, port)
	case "volumes":
		if value.Subfield != "" {
			return fmt.Errorf(`cannot extend "volumes" with sub-field`)
		}
		var volume WorkloadVolume
		if err := volume.Set(value.Value); err != nil {
			return err
		}
		w.Volumes = append(w.Volumes, volume)
	case "env":
		if value.Subfield == "" {
			return fmt.Errorf(`cannot extend "env" without sub-field`)
		}
		if w.EnvVars == nil {
			w.EnvVars = make(map[string]string)
		} else if _, ok := w.EnvVars[value.Subfield]; ok {
			return fmt.Errorf(`"env" field %q already set`, value.Subfield)
		}
		w.EnvVars[value.Subfield] = value.Value
	default:
		return fmt.Errorf("unrecognized field %q", value.Field)
	}
	return nil
}

// Apply makes a copy of the Workload and applies the given overrides
// and additions to that copy.
func (w *Workload) Apply(overrides []WorkloadFieldValue, additions []WorkloadFieldValue) (*Workload, error) {
	workload := w.Copy()
	for _, value := range overrides {
		if err := workload.Override(value); err != nil {
			return nil, err
		}
	}
	for _, value := range additions {
		if err := workload.Extend(value); err != nil {
			return nil, err
		}
	}
	return &workload, nil
}

// Validate checks the Workload for errors.
func (w Workload) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("missing name")
	}
	if w.Type == "" {
		return fmt.Errorf("metadata: workloads.%s.type: name is required", w.Name)
	}

	if err := w.validatePorts(); err != nil {
		return err
	}

	if err := w.validateStorage(); err != nil {
		return err
	}

	return nil
}

func (w Workload) validatePorts() error {
	for _, port := range w.Ports {
		if port.External < 0 {
			return fmt.Errorf("metadata: workloads.%s.ports: specified endpoint %q unknown for %v", w.Name, port.Endpoint, port)
		}
	}
	return nil
}

func (w Workload) validateStorage() error {
	for _, volume := range w.Volumes {
		if volume.Name != "" && volume.ExternalMount == "" {
			if volume.storage == nil {
				return fmt.Errorf("metadata: workloads.%s.volumes: specified storage %q unknown for %v", w.Name, volume.Name, volume)
			}
			if volume.storage.Type != StorageFilesystem {
				return fmt.Errorf("metadata: workloads.%s.volumes: linked storage %q must be filesystem for %v", w.Name, volume.Name, volume)
			}
			if volume.storage.Location == "" {
				return fmt.Errorf("metadata: workloads.%s.volumes: linked storage %q missing location for %v", w.Name, volume.Name, volume)
			}
		}
	}
	return nil
}

// WorkloadPort is network port information for a workload workload.
type WorkloadPort struct {
	// External is the port on the host.
	External int
	// Internal is the port on the workload.
	Internal int
	// Endpoint is the unit-relation endpoint matching the external
	// port, if any.
	Endpoint string
}

// Set parses the provided string and sets the appropriate fields.
func (w *WorkloadPort) Set(raw string) error {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid value %q", raw)
	}
	if err := w.SetExternal(parts[0]); err != nil {
		return err
	}
	if err := w.SetInternal(parts[1]); err != nil {
		return err
	}
	return nil
}

// SetExternal parses the provided string and sets the appropriate fields.
func (w *WorkloadPort) SetExternal(portStr string) error {
	w.External = 0
	w.Endpoint = ""
	if strings.HasPrefix(portStr, "<") && strings.HasSuffix(portStr, ">") {
		// The port was specified by a relation endpoint rather than a
		// port number.
		w.Endpoint = portStr[1 : len(portStr)-1]
	} else {
		// It's just a port number.
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("expected int got %q", portStr)
		}
		w.External = port
	}
	return nil
}

// SetInternal parses the provided string and sets the appropriate fields.
func (w *WorkloadPort) SetInternal(portStr string) error {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("expected int got %q", portStr)
	}
	w.Internal = port
	return nil
}

// WorkloadVolume is storage volume information for a workload workload.
type WorkloadVolume struct {
	// ExternalMount is the path on the host.
	ExternalMount string
	// InternalMount is the path on the workload.
	InternalMount string
	// Mode is the "ro" OR "rw"
	Mode string
	// Name is the name of the storage metadata entry, if any.
	Name string

	// storage is the storage that matched the Storage field.
	storage *Storage
}

// Copy create a deep copy of the WorkloadVolume.
func (copied WorkloadVolume) Copy() WorkloadVolume {
	copied.storage = nil
	return copied
}

// Set parses the provided string and sets the appropriate fields.
func (pv *WorkloadVolume) Set(raw string) error {
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
func (pv *WorkloadVolume) SetExternal(volume string) {
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
func (pv *WorkloadVolume) SetInternal(volume string) {
	pv.InternalMount = volume
}

// SetMode parses the provided string and sets the appropriate fields.
func (pv *WorkloadVolume) SetMode(mode string) error {
	if _, err := schema.OneOf(schema.Const("rw"), schema.Const("ro")).Coerce(mode, nil); err != nil {
		return fmt.Errorf(`expected "rw" or "ro" for mode, got %q`, mode)
	}
	pv.Mode = mode
	return nil
}

func parseWorkloads(data interface{}, provides map[string]Relation, storage map[string]Storage) map[string]Workload {
	if data == nil {
		return nil
	}
	result := make(map[string]Workload)
	for name, workloadData := range data.(map[string]interface{}) {
		workloadMap := workloadData.(map[string]interface{})
		result[name] = parseWorkload(name, workloadMap, provides, storage)
	}
	return result
}

func parseWorkload(name string, coerced map[string]interface{}, provides map[string]Relation, storage map[string]Storage) Workload {
	workload := Workload{
		Name: name,
		Type: coerced["type"].(string),
	}

	if description, ok := coerced["description"]; ok {
		workload.Description = description.(string)
	}

	if typeMap, ok := coerced["type-options"]; ok {
		options := typeMap.(map[string]interface{})
		if len(options) > 0 {
			workload.TypeOptions = make(map[string]string)
			for k, v := range options {
				workload.TypeOptions[k] = v.(string)
			}
		}
	}

	if command, ok := coerced["command"]; ok {
		workload.Command = command.(string)
	}

	if image, ok := coerced["image"]; ok {
		workload.Image = image.(string)
	}

	if portsList, ok := coerced["ports"]; ok {
		for _, portRaw := range portsList.([]interface{}) {
			port := portRaw.(*WorkloadPort)
			if port.External == 0 {
				port.External = -1
				for endpoint := range provides {
					if port.Endpoint == endpoint {
						port.External = 0
						break
					}
				}
			}
			workload.Ports = append(workload.Ports, *port)
		}
	}

	if volumeList, ok := coerced["volumes"]; ok {
		for _, volumeRaw := range volumeList.([]interface{}) {
			volume := *volumeRaw.(*WorkloadVolume)
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
			workload.Volumes = append(workload.Volumes, volume)
		}
	}

	if envMap, ok := coerced["env"]; ok {
		workload.EnvVars = make(map[string]string)
		for k, v := range envMap.(map[string]interface{}) {
			workload.EnvVars[k] = v.(string)
		}
	}

	return workload
}

func checkWorkloads(workloads map[string]Workload) error {
	for _, workload := range workloads {
		if err := workload.Validate(); err != nil {
			return err
		}
	}
	return nil
}

var workloadSchema = schema.FieldMap(
	schema.Fields{
		"description":  schema.String(),
		"type":         schema.String(),
		"type-options": schema.StringMap(schema.Stringified()),
		"command":      schema.String(),
		"image":        schema.String(),
		"ports":        schema.List(workloadPortsChecker{}),
		"volumes":      schema.List(workloadVolumeChecker{}),
		"env":          schema.StringMap(schema.Stringified()),
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

type workloadPortsChecker struct{}

// Coerce implements schema.Checker.
func (c workloadPortsChecker) Coerce(v interface{}, path []string) (interface{}, error) {
	if _, err := schema.String().Coerce(v, path); err != nil {
		return nil, err
	}
	item := v.(string)

	var port WorkloadPort
	if err := port.Set(item); err != nil {
		return nil, fmt.Errorf("%s: %v", strings.Join(path[1:], ""), err)
	}
	return &port, nil
}

type workloadVolumeChecker struct{}

// Coerce implements schema.Checker.
func (c workloadVolumeChecker) Coerce(v interface{}, path []string) (interface{}, error) {
	if _, err := schema.String().Coerce(v, path); err != nil {
		return nil, err
	}
	item := v.(string)

	var volume WorkloadVolume
	if err := volume.Set(item); err != nil {
		return nil, fmt.Errorf("%s: %v", strings.Join(path[1:], ""), err)
	}
	return &volume, nil
}
