// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/juju/schema"
	"github.com/juju/utils"
	"gopkg.in/yaml.v2"

	"gopkg.in/juju/charm.v6-unstable/hooks"
)

// RelationScope describes the scope of a relation.
type RelationScope string

// Note that schema doesn't support custom string types,
// so when we use these values in a schema.Checker,
// we must store them as strings, not RelationScopes.

const (
	ScopeGlobal    RelationScope = "global"
	ScopeContainer RelationScope = "container"
)

// RelationRole defines the role of a relation.
type RelationRole string

const (
	RoleProvider RelationRole = "provider"
	RoleRequirer RelationRole = "requirer"
	RolePeer     RelationRole = "peer"
)

// StorageType defines a storage type.
type StorageType string

const (
	StorageBlock      StorageType = "block"
	StorageFilesystem StorageType = "filesystem"
)

// Storage represents a charm's storage requirement.
type Storage struct {
	// Name is the name of the store.
	//
	// Name has no default, and must be specified.
	Name string `bson:"name"`

	// Description is a description of the store.
	//
	// Description has no default, and is optional.
	Description string `bson:"description"`

	// Type is the storage type: filesystem or block-device.
	//
	// Type has no default, and must be specified.
	Type StorageType `bson:"type"`

	// Shared indicates that the storage is shared between all units of
	// a service deployed from the charm. It is an error to attempt to
	// assign non-shareable storage to a "shared" storage requirement.
	//
	// Shared defaults to false.
	Shared bool `bson:"shared"`

	// ReadOnly indicates that the storage should be made read-only if
	// possible. If the storage cannot be made read-only, Juju will warn
	// the user.
	//
	// ReadOnly defaults to false.
	ReadOnly bool `bson:"read-only"`

	// CountMin is the number of storage instances that must be attached
	// to the charm for it to be useful; the charm will not install until
	// this number has been satisfied. This must be a non-negative number.
	//
	// CountMin defaults to 1 for singleton stores.
	CountMin int `bson:"countmin"`

	// CountMax is the largest number of storage instances that can be
	// attached to the charm. If CountMax is -1, then there is no upper
	// bound.
	//
	// CountMax defaults to 1 for singleton stores.
	CountMax int `bson:"countmax"`

	// MinimumSize is the minimum size of store that the charm needs to
	// work at all. This is not a recommended size or a comfortable size
	// or a will-work-well size, just a bare minimum below which the charm
	// is going to break.
	// MinimumSize requires a unit, one of MGTPEZY, and is stored as MiB.
	//
	// There is no default MinimumSize; if left unspecified, a provider
	// specific default will be used, typically 1GB for block storage.
	MinimumSize uint64 `bson:"minimum-size"`

	// Location is the mount location for filesystem stores. For multi-
	// stores, the location acts as the parent directory for each mounted
	// store.
	//
	// Location has no default, and is optional.
	Location string `bson:"location,omitempty"`

	// Properties allow the charm author to characterise the relative storage
	// performance requirements and sensitivities for each store.
	// eg “transient” is used to indicate that non persistent storage is acceptable,
	// such as tmpfs or ephemeral instance disks.
	//
	// Properties has no default, and is optional.
	Properties []string `bson:"properties,omitempty"`
}

// Relation represents a single relation defined in the charm
// metadata.yaml file.
type Relation struct {
	Name      string        `bson:"name"`
	Role      RelationRole  `bson:"role"`
	Interface string        `bson:"interface"`
	Optional  bool          `bson:"optional"`
	Limit     int           `bson:"limit"`
	Scope     RelationScope `bson:"scope"`
}

// ImplementedBy returns whether the relation is implemented by the supplied charm.
func (r Relation) ImplementedBy(ch Charm) bool {
	if r.IsImplicit() {
		return true
	}
	var m map[string]Relation
	switch r.Role {
	case RoleProvider:
		m = ch.Meta().Provides
	case RoleRequirer:
		m = ch.Meta().Requires
	case RolePeer:
		m = ch.Meta().Peers
	default:
		panic(fmt.Errorf("unknown relation role %q", r.Role))
	}
	rel, found := m[r.Name]
	if !found {
		return false
	}
	if rel.Interface == r.Interface {
		switch r.Scope {
		case ScopeGlobal:
			return rel.Scope != ScopeContainer
		case ScopeContainer:
			return true
		default:
			panic(fmt.Errorf("unknown relation scope %q", r.Scope))
		}
	}
	return false
}

// IsImplicit returns whether the relation is supplied by juju itself,
// rather than by a charm.
func (r Relation) IsImplicit() bool {
	return (r.Name == "juju-info" &&
		r.Interface == "juju-info" &&
		r.Role == RoleProvider)
}

// Meta represents all the known content that may be defined
// within a charm's metadata.yaml file.
// Note: Series is serialised for backward compatibility
// as "supported-series" because a previous
// charm version had an incompatible Series field that
// was unused in practice but still serialized. This
// only applies to JSON because Meta has a custom
// YAML marshaller.
type Meta struct {
	Name           string                  `bson:"name" json:"name" yaml:"name"`
	Summary        string                  `bson:"summary" json:"summary" yaml:"summary"`
	Description    string                  `bson:"description" json:"description" yaml:"description"`
	Subordinate    bool                    `bson:"subordinate" json:"subordinate" yaml:"subordinate,omitempty"`
	Provides       map[string]Relation     `bson:"provides,omitempty" json:"provides,omitempty" yaml:"provides,omitempty"`
	Requires       map[string]Relation     `bson:"requires,omitempty" json:"requires,omitempty" yaml:"requires,omitempty"`
	Peers          map[string]Relation     `bson:"peers,omitempty" json:"peers,omitempty" yaml:"peers,omitempty"`
	Format         int                     `bson:"format,omitempty" json:"format,omitempty" yaml:"-"`
	OldRevision    int                     `bson:"oldrevision,omitempty" yaml:"-"` // Obsolete
	Categories     []string                `bson:"categories,omitempty" json:"categories,omitempty" yaml:"categories,omitempty"`
	Tags           []string                `bson:"tags,omitempty" json:"tag,omitempty" yaml:"tags,omitempty"`
	Series         []string                `bson:"series,omitempty" json:"supported-series,omitempty" yaml:"series,omitempty"`
	Storage        map[string]Storage      `bson:"storage,omitempty" json:"storage,omitempty" yaml:"-"`
	PayloadClasses map[string]PayloadClass `bson:"payloadclasses,omitempty" json:"payloadclasses,omitempty" yaml:"-"`
}

func generateRelationHooks(relName string, allHooks map[string]bool) {
	for _, hookName := range hooks.RelationHooks() {
		allHooks[fmt.Sprintf("%s-%s", relName, hookName)] = true
	}
}

// Hooks returns a map of all possible valid hooks, taking relations
// into account. It's a map to enable fast lookups, and the value is
// always true.
func (m Meta) Hooks() map[string]bool {
	allHooks := make(map[string]bool)
	// Unit hooks
	for _, hookName := range hooks.UnitHooks() {
		allHooks[string(hookName)] = true
	}
	// Relation hooks
	for hookName := range m.Provides {
		generateRelationHooks(hookName, allHooks)
	}
	for hookName := range m.Requires {
		generateRelationHooks(hookName, allHooks)
	}
	for hookName := range m.Peers {
		generateRelationHooks(hookName, allHooks)
	}
	return allHooks
}

// Used for parsing Categories and Tags.
func parseStringList(list interface{}) []string {
	if list == nil {
		return nil
	}
	slice := list.([]interface{})
	result := make([]string, 0, len(slice))
	for _, elem := range slice {
		result = append(result, elem.(string))
	}
	return result
}

// ReadMeta reads the content of a metadata.yaml file and returns
// its representation.
func ReadMeta(r io.Reader) (meta *Meta, err error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}
	raw := make(map[interface{}]interface{})
	err = yaml.Unmarshal(data, raw)
	if err != nil {
		return
	}
	v, err := charmSchema.Coerce(raw, nil)
	if err != nil {
		return nil, errors.New("metadata: " + err.Error())
	}
	m := v.(map[string]interface{})
	meta = &Meta{}
	meta.Name = m["name"].(string)
	// Schema decodes as int64, but the int range should be good
	// enough for revisions.
	meta.Summary = m["summary"].(string)
	meta.Description = m["description"].(string)
	meta.Provides = parseRelations(m["provides"], RoleProvider)
	meta.Requires = parseRelations(m["requires"], RoleRequirer)
	meta.Peers = parseRelations(m["peers"], RolePeer)
	meta.Format = int(m["format"].(int64))
	meta.Categories = parseStringList(m["categories"])
	meta.Tags = parseStringList(m["tags"])
	if subordinate := m["subordinate"]; subordinate != nil {
		meta.Subordinate = subordinate.(bool)
	}
	if rev := m["revision"]; rev != nil {
		// Obsolete
		meta.OldRevision = int(m["revision"].(int64))
	}
	meta.Series = parseStringList(m["series"])
	meta.Storage = parseStorage(m["storage"])
	meta.PayloadClasses = parsePayloadClasses(m["payloads"])
	if err := meta.Check(); err != nil {
		return nil, err
	}
	return meta, nil
}

// MarshalYAML implements yaml.Marshaler.MarshalYAML()
func (r Relation) MarshalYAML() (interface{}, error) {
	// See calls to ifaceExpander in charmSchema.
	noLimit := 1
	if r.Role == RoleProvider {
		noLimit = 0
	}

	if !r.Optional && r.Limit == noLimit && r.Scope == ScopeGlobal {
		// All attributes are default, so use the simple string form of the relation.
		return r.Interface, nil
	}
	mr := struct {
		Interface string        `yaml:"interface"`
		Limit     *int          `yaml:"limit,omitempty"`
		Optional  bool          `yaml:"optional,omitempty"`
		Scope     RelationScope `yaml:"scope,omitempty"`
	}{
		Interface: r.Interface,
		Optional:  r.Optional,
	}
	if r.Limit != noLimit {
		mr.Limit = &r.Limit
	}
	if r.Scope != ScopeGlobal {
		mr.Scope = r.Scope
	}
	return mr, nil
}

// Check checks that the metadata is well-formed.
func (meta Meta) Check() error {
	// Check for duplicate or forbidden relation names or interfaces.
	names := map[string]bool{}
	checkRelations := func(src map[string]Relation, role RelationRole) error {
		for name, rel := range src {
			if rel.Name != name {
				return fmt.Errorf("charm %q has mismatched relation name %q; expected %q", meta.Name, rel.Name, name)
			}
			if rel.Role != role {
				return fmt.Errorf("charm %q has mismatched role %q; expected %q", meta.Name, rel.Role, role)
			}
			// Container-scoped require relations on subordinates are allowed
			// to use the otherwise-reserved juju-* namespace.
			if !meta.Subordinate || role != RoleRequirer || rel.Scope != ScopeContainer {
				if reservedName(name) {
					return fmt.Errorf("charm %q using a reserved relation name: %q", meta.Name, name)
				}
			}
			if role != RoleRequirer {
				if reservedName(rel.Interface) {
					return fmt.Errorf("charm %q relation %q using a reserved interface: %q", meta.Name, name, rel.Interface)
				}
			}
			if names[name] {
				return fmt.Errorf("charm %q using a duplicated relation name: %q", meta.Name, name)
			}
			names[name] = true
		}
		return nil
	}
	if err := checkRelations(meta.Provides, RoleProvider); err != nil {
		return err
	}
	if err := checkRelations(meta.Requires, RoleRequirer); err != nil {
		return err
	}
	if err := checkRelations(meta.Peers, RolePeer); err != nil {
		return err
	}

	// Subordinate charms must have at least one relation that
	// has container scope, otherwise they can't relate to the
	// principal.
	if meta.Subordinate {
		valid := false
		if meta.Requires != nil {
			for _, relationData := range meta.Requires {
				if relationData.Scope == ScopeContainer {
					valid = true
					break
				}
			}
		}
		if !valid {
			return fmt.Errorf("subordinate charm %q lacks \"requires\" relation with container scope", meta.Name)
		}
	}

	for _, series := range meta.Series {
		if !IsValidSeries(series) {
			return fmt.Errorf("charm %q declares invalid series: %q", meta.Name, series)
		}
	}

	names = make(map[string]bool)
	for name, store := range meta.Storage {
		if store.Location != "" && store.Type != StorageFilesystem {
			return fmt.Errorf(`charm %q storage %q: location may not be specified for "type: %s"`, meta.Name, name, store.Type)
		}
		if store.Type == "" {
			return fmt.Errorf("charm %q storage %q: type must be specified", meta.Name, name)
		}
		if store.CountMin < 0 {
			return fmt.Errorf("charm %q storage %q: invalid minimum count %d", meta.Name, name, store.CountMin)
		}
		if store.CountMax == 0 || store.CountMax < -1 {
			return fmt.Errorf("charm %q storage %q: invalid maximum count %d", meta.Name, name, store.CountMax)
		}
		if names[name] {
			return fmt.Errorf("charm %q storage %q: duplicated storage name", meta.Name, name)
		}
		names[name] = true
	}

	for name, payloadClass := range meta.PayloadClasses {
		if payloadClass.Name != name {
			return fmt.Errorf("mismatch on payload class name (%q != %q)", payloadClass.Name, name)
		}
		if err := payloadClass.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func reservedName(name string) bool {
	return name == "juju" || strings.HasPrefix(name, "juju-")
}

func parseRelations(relations interface{}, role RelationRole) map[string]Relation {
	if relations == nil {
		return nil
	}
	result := make(map[string]Relation)
	for name, rel := range relations.(map[string]interface{}) {
		relMap := rel.(map[string]interface{})
		relation := Relation{
			Name:      name,
			Role:      role,
			Interface: relMap["interface"].(string),
			Optional:  relMap["optional"].(bool),
		}
		if scope := relMap["scope"]; scope != nil {
			relation.Scope = RelationScope(scope.(string))
		}
		if relMap["limit"] != nil {
			// Schema defaults to int64, but we know
			// the int range should be more than enough.
			relation.Limit = int(relMap["limit"].(int64))
		}
		result[name] = relation
	}
	return result
}

// Schema coercer that expands the interface shorthand notation.
// A consistent format is easier to work with than considering the
// potential difference everywhere.
//
// Supports the following variants::
//
//   provides:
//     server: riak
//     admin: http
//     foobar:
//       interface: blah
//
//   provides:
//     server:
//       interface: mysql
//       limit:
//       optional: false
//
// In all input cases, the output is the fully specified interface
// representation as seen in the mysql interface description above.
func ifaceExpander(limit interface{}) schema.Checker {
	return ifaceExpC{limit}
}

type ifaceExpC struct {
	limit interface{}
}

var (
	stringC = schema.String()
	mapC    = schema.StringMap(schema.Any())
)

func (c ifaceExpC) Coerce(v interface{}, path []string) (newv interface{}, err error) {
	s, err := stringC.Coerce(v, path)
	if err == nil {
		newv = map[string]interface{}{
			"interface": s,
			"limit":     c.limit,
			"optional":  false,
			"scope":     string(ScopeGlobal),
		}
		return
	}

	v, err = mapC.Coerce(v, path)
	if err != nil {
		return
	}
	m := v.(map[string]interface{})
	if _, ok := m["limit"]; !ok {
		m["limit"] = c.limit
	}
	return ifaceSchema.Coerce(m, path)
}

var ifaceSchema = schema.FieldMap(
	schema.Fields{
		"interface": schema.String(),
		"limit":     schema.OneOf(schema.Const(nil), schema.Int()),
		"scope":     schema.OneOf(schema.Const(string(ScopeGlobal)), schema.Const(string(ScopeContainer))),
		"optional":  schema.Bool(),
	},
	schema.Defaults{
		"scope":    string(ScopeGlobal),
		"optional": false,
	},
)

func parseStorage(stores interface{}) map[string]Storage {
	if stores == nil {
		return nil
	}
	result := make(map[string]Storage)
	for name, store := range stores.(map[string]interface{}) {
		storeMap := store.(map[string]interface{})
		store := Storage{
			Name:     name,
			Type:     StorageType(storeMap["type"].(string)),
			Shared:   storeMap["shared"].(bool),
			ReadOnly: storeMap["read-only"].(bool),
			CountMin: 1,
			CountMax: 1,
		}
		if desc, ok := storeMap["description"].(string); ok {
			store.Description = desc
		}
		if multiple, ok := storeMap["multiple"].(map[string]interface{}); ok {
			if r, ok := multiple["range"].([2]int); ok {
				store.CountMin, store.CountMax = r[0], r[1]
			}
		}
		if minSize, ok := storeMap["minimum-size"].(uint64); ok {
			store.MinimumSize = minSize
		}
		if loc, ok := storeMap["location"].(string); ok {
			store.Location = loc
		}
		if properties, ok := storeMap["properties"].([]interface{}); ok {
			for _, p := range properties {
				store.Properties = append(store.Properties, p.(string))
			}
		}
		result[name] = store
	}
	return result
}

var storageSchema = schema.FieldMap(
	schema.Fields{
		"type":      schema.OneOf(schema.Const(string(StorageBlock)), schema.Const(string(StorageFilesystem))),
		"shared":    schema.Bool(),
		"read-only": schema.Bool(),
		"multiple": schema.FieldMap(
			schema.Fields{
				"range": storageCountC{}, // m, m-n, m+, m-
			},
			schema.Defaults{},
		),
		"minimum-size": storageSizeC{},
		"location":     schema.String(),
		"description":  schema.String(),
		"properties":   schema.List(propertiesC{}),
	},
	schema.Defaults{
		"shared":       false,
		"read-only":    false,
		"multiple":     schema.Omit,
		"location":     schema.Omit,
		"description":  schema.Omit,
		"properties":   schema.Omit,
		"minimum-size": schema.Omit,
	},
)

type storageCountC struct{}

var storageCountRE = regexp.MustCompile("^([0-9]+)([-+]|-[0-9]+)$")

func (c storageCountC) Coerce(v interface{}, path []string) (newv interface{}, err error) {
	s, err := schema.OneOf(schema.Int(), stringC).Coerce(v, path)
	if err != nil {
		return nil, err
	}
	if m, ok := s.(int64); ok {
		// We've got a count of the form "m": m represents
		// both the minimum and maximum.
		if m <= 0 {
			return nil, fmt.Errorf("%s: invalid count %v", strings.Join(path[1:], ""), m)
		}
		return [2]int{int(m), int(m)}, nil
	}
	match := storageCountRE.FindStringSubmatch(s.(string))
	if match == nil {
		return nil, fmt.Errorf("%s: value %q does not match 'm', 'm-n', or 'm+'", strings.Join(path[1:], ""), s)
	}
	var m, n int
	if m, err = strconv.Atoi(match[1]); err != nil {
		return nil, err
	}
	if len(match[2]) == 1 {
		// We've got a count of the form "m+" or "m-":
		// m represents the minimum, and there is no
		// upper bound.
		n = -1
	} else {
		if n, err = strconv.Atoi(match[2][1:]); err != nil {
			return nil, err
		}
	}
	return [2]int{m, n}, nil
}

type storageSizeC struct{}

func (c storageSizeC) Coerce(v interface{}, path []string) (newv interface{}, err error) {
	s, err := schema.String().Coerce(v, path)
	if err != nil {
		return nil, err
	}
	return utils.ParseSize(s.(string))
}

type propertiesC struct{}

func (c propertiesC) Coerce(v interface{}, path []string) (newv interface{}, err error) {
	return schema.OneOf(schema.Const("transient")).Coerce(v, path)
}

var charmSchema = schema.FieldMap(
	schema.Fields{
		"name":        schema.String(),
		"summary":     schema.String(),
		"description": schema.String(),
		"peers":       schema.StringMap(ifaceExpander(int64(1))),
		"provides":    schema.StringMap(ifaceExpander(nil)),
		"requires":    schema.StringMap(ifaceExpander(int64(1))),
		"revision":    schema.Int(), // Obsolete
		"format":      schema.Int(),
		"subordinate": schema.Bool(),
		"categories":  schema.List(schema.String()),
		"tags":        schema.List(schema.String()),
		"series":      schema.List(schema.String()),
		"storage":     schema.StringMap(storageSchema),
		"payloads":    schema.StringMap(payloadClassSchema),
	},
	schema.Defaults{
		"provides":    schema.Omit,
		"requires":    schema.Omit,
		"peers":       schema.Omit,
		"revision":    schema.Omit,
		"format":      1,
		"subordinate": schema.Omit,
		"categories":  schema.Omit,
		"tags":        schema.Omit,
		"series":      schema.Omit,
		"storage":     schema.Omit,
		"payloads":    schema.Omit,
	},
)
