// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/juju/collections/set"
	"github.com/juju/errors"
	"github.com/juju/names/v4"
	"github.com/juju/os/v2"
	"github.com/juju/os/v2/series"
	"github.com/juju/schema"
	"github.com/juju/systems"
	"github.com/juju/systems/channel"
	"github.com/juju/utils/v2"
	"github.com/juju/version"
	"gopkg.in/yaml.v2"

	"github.com/juju/charm/v9/hooks"
	"github.com/juju/charm/v9/resource"
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
	// an application deployed from the charm. It is an error to attempt to
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

// DeviceType defines a device type.
type DeviceType string

// Device represents a charm's device requirement (GPU for example).
type Device struct {
	// Name is the name of the device.
	Name string `bson:"name"`

	// Description is a description of the device.
	Description string `bson:"description"`

	// Type is the device type.
	// currently supported types are
	// - gpu
	// - nvidia.com/gpu
	// - amd.com/gpu
	Type DeviceType `bson:"type"`

	// CountMin is the min number of devices that the charm requires.
	CountMin int64 `bson:"countmin"`

	// CountMax is the max number of devices that the charm requires.
	CountMax int64 `bson:"countmax"`
}

// DeploymentType defines a deployment type.
type DeploymentType string

const (
	DeploymentStateless DeploymentType = "stateless"
	DeploymentStateful  DeploymentType = "stateful"
	DeploymentDaemon    DeploymentType = "daemon"
)

// DeploymentMode defines a deployment mode.
type DeploymentMode string

const (
	ModeOperator DeploymentMode = "operator"
	ModeWorkload DeploymentMode = "workload"
)

// ServiceType defines a service type.
type ServiceType string

const (
	ServiceCluster      ServiceType = "cluster"
	ServiceLoadBalancer ServiceType = "loadbalancer"
	ServiceExternal     ServiceType = "external"
	ServiceOmit         ServiceType = "omit"
)

var validServiceTypes = map[os.OSType][]ServiceType{
	os.Kubernetes: {
		ServiceCluster,
		ServiceLoadBalancer,
		ServiceExternal,
		ServiceOmit,
	},
}

// Deployment represents a charm's deployment requirements in the charm
// metadata.yaml file.
type Deployment struct {
	DeploymentType DeploymentType `bson:"type"`
	DeploymentMode DeploymentMode `bson:"mode"`
	ServiceType    ServiceType    `bson:"service"`
	MinVersion     string         `bson:"min-version"`
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
		panic(errors.Errorf("unknown relation role %q", r.Role))
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
			panic(errors.Errorf("unknown relation scope %q", r.Scope))
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
	Name           string                   `bson:"name" json:"Name"`
	Summary        string                   `bson:"summary" json:"Summary"`
	Description    string                   `bson:"description" json:"Description"`
	Subordinate    bool                     `bson:"subordinate" json:"Subordinate"`
	Provides       map[string]Relation      `bson:"provides,omitempty" json:"Provides,omitempty"`
	Requires       map[string]Relation      `bson:"requires,omitempty" json:"Requires,omitempty"`
	Peers          map[string]Relation      `bson:"peers,omitempty" json:"Peers,omitempty"`
	ExtraBindings  map[string]ExtraBinding  `bson:"extra-bindings,omitempty" json:"ExtraBindings,omitempty"`
	Categories     []string                 `bson:"categories,omitempty" json:"Categories,omitempty"`
	Tags           []string                 `bson:"tags,omitempty" json:"Tags,omitempty"`
	Series         []string                 `bson:"series,omitempty" json:"SupportedSeries,omitempty"`
	Storage        map[string]Storage       `bson:"storage,omitempty" json:"Storage,omitempty"`
	Devices        map[string]Device        `bson:"devices,omitempty" json:"Devices,omitempty"`
	Deployment     *Deployment              `bson:"deployment,omitempty" json:"Deployment,omitempty"`
	PayloadClasses map[string]PayloadClass  `bson:"payloadclasses,omitempty" json:"PayloadClasses,omitempty"`
	Resources      map[string]resource.Meta `bson:"resources,omitempty" json:"Resources,omitempty"`
	Terms          []string                 `bson:"terms,omitempty" json:"Terms,omitempty"`
	MinJujuVersion version.Number           `bson:"min-juju-version,omitempty" json:"min-juju-version,omitempty"`

	Systems       []systems.System     `bson:"systems,omitempty" json:"systems,omitempty" yaml:"systems,omitempty"`
	Platforms     []Platform           `bson:"platforms,omitempty" json:"platforms,omitempty" yaml:"platforms,omitempty"`
	Architectures []Architecture       `bson:"architectures,omitempty" json:"architectures,omitempty" yaml:"architectures,omitempty"`
	Containers    map[string]Container `bson:"containers,omitempty" json:"containers,omitempty" yaml:"containers,omitempty"`
}

// Platform describes deployment plaforms charms can be deployed to.
// NOTE: for v2 charms only.
type Platform string

// Platforms v2 charms support.
const (
	PlatformMachine    Platform = "machine"
	PlatformKubernetes Platform = "kubernetes"
)

// Architecture describes architectures charms can be deployed to.
// NOTE: for v2 charms only.
type Architecture string

// Architectures v2 charms support.
const (
	AMD64   Architecture = "amd64"
	ARM64   Architecture = "arm64"
	PPC64EL Architecture = "ppc64el"
	S390X   Architecture = "s390x"
)

// Container specifies the possible systems it supports and mounts it wants.
type Container struct {
	Systems []systems.System `bson:"systems,omitempty" json:"systems,omitempty" yaml:"systems,omitempty"`
	Mounts  []Mount          `bson:"mounts,omitempty" json:"mounts,omitempty" yaml:"mounts,omitempty"`
}

// Mount allows a container to mount a storage filesystem from the storage top-level directive.
type Mount struct {
	Storage  string `bson:"storage,omitempty" json:"storage,omitempty" yaml:"storage,omitempty"`
	Location string `bson:"location,omitempty" json:"location,omitempty" yaml:"location,omitempty"`
}

// Format of the parsed charm.
type Format int

// Formats are the different versions of charm metadata supported.
const (
	FormatV1 = iota
	FormatV2 = iota
)

func generateRelationHooks(relName string, allHooks map[string]bool) {
	for _, hookName := range hooks.RelationHooks() {
		allHooks[fmt.Sprintf("%s-%s", relName, hookName)] = true
	}
}

func generateContainerHooks(containerName string, allHooks map[string]bool) {
	// Containers using pebble trigger workload hooks.
	for _, hookName := range hooks.WorkloadHooks() {
		allHooks[fmt.Sprintf("%s-%s", containerName, hookName)] = true
	}
}

func generateStorageHooks(storageName string, allHooks map[string]bool) {
	for _, hookName := range hooks.StorageHooks() {
		allHooks[fmt.Sprintf("%s-%s", storageName, hookName)] = true
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
	for storageName := range m.Storage {
		generateStorageHooks(storageName, allHooks)
	}
	for containerName := range m.Containers {
		generateContainerHooks(containerName, allHooks)
	}
	return allHooks
}

// Format returns the charm metadata format version.
// Charms that specify systems are v2. Otherwise it
// defaults to v1.
func (m Meta) Format() Format {
	if m.Systems != nil {
		return FormatV2
	}
	return FormatV1
}

// ComputedSeries of a charm. This is to support legacy logic on new
// charms that use Systems.
func (m Meta) ComputedSeries() []string {
	if m.Format() == FormatV1 {
		return m.Series
	}
	// The slice must be ordered based on system appearance but
	// have unique elements.
	seriesSlice := []string(nil)
	seriesSet := set.NewStrings()
	for _, system := range m.Systems {
		series := system.String()
		if !seriesSet.Contains(series) {
			seriesSet.Add(series)
			seriesSlice = append(seriesSlice, series)
		}
	}
	return seriesSlice
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

var validTermName = regexp.MustCompile(`^[a-z](-?[a-z0-9]+)+$`)

// TermsId represents a single term id. The term can either be owned
// or "public" (meaning there is no owner).
// The Revision starts at 1. Therefore a value of 0 means the revision
// is unset.
type TermsId struct {
	Tenant   string
	Owner    string
	Name     string
	Revision int
}

// Validate returns an error if the Term contains invalid data.
func (t *TermsId) Validate() error {
	if t.Tenant != "" && t.Tenant != "cs" {
		if !validTermName.MatchString(t.Tenant) {
			return errors.Errorf("wrong term tenant format %q", t.Tenant)
		}
	}
	if t.Owner != "" && !names.IsValidUser(t.Owner) {
		return errors.Errorf("wrong owner format %q", t.Owner)
	}
	if !validTermName.MatchString(t.Name) {
		return errors.Errorf("wrong term name format %q", t.Name)
	}
	if t.Revision < 0 {
		return errors.Errorf("negative term revision")
	}
	return nil
}

// String returns the term in canonical form.
// This would be one of:
//   tenant:owner/name/revision
//   tenant:name
//   owner/name/revision
//   owner/name
//   name/revision
//   name
func (t *TermsId) String() string {
	id := make([]byte, 0, len(t.Tenant)+1+len(t.Owner)+1+len(t.Name)+4)
	if t.Tenant != "" {
		id = append(id, t.Tenant...)
		id = append(id, ':')
	}
	if t.Owner != "" {
		id = append(id, t.Owner...)
		id = append(id, '/')
	}
	id = append(id, t.Name...)
	if t.Revision != 0 {
		id = append(id, '/')
		id = strconv.AppendInt(id, int64(t.Revision), 10)
	}
	return string(id)
}

// ParseTerm takes a termID as a string and parses it into a Term.
// A complete term is in the form:
// tenant:owner/name/revision
// This function accepts partially specified identifiers
// typically in one of the following forms:
// name
// owner/name
// owner/name/27 # Revision 27
// name/283 # Revision 283
// cs:owner/name # Tenant cs
func ParseTerm(s string) (*TermsId, error) {
	tenant := ""
	termid := s
	if t := strings.SplitN(s, ":", 2); len(t) == 2 {
		tenant = t[0]
		termid = t[1]
	}

	tokens := strings.Split(termid, "/")
	var term TermsId
	switch len(tokens) {
	case 1: // "name"
		term = TermsId{
			Tenant: tenant,
			Name:   tokens[0],
		}
	case 2: // owner/name or name/123
		termRevision, err := strconv.Atoi(tokens[1])
		if err != nil { // owner/name
			term = TermsId{
				Tenant: tenant,
				Owner:  tokens[0],
				Name:   tokens[1],
			}
		} else { // name/123
			term = TermsId{
				Tenant:   tenant,
				Name:     tokens[0],
				Revision: termRevision,
			}
		}
	case 3: // owner/name/123
		termRevision, err := strconv.Atoi(tokens[2])
		if err != nil {
			return nil, errors.Errorf("invalid revision number %q %v", tokens[2], err)
		}
		term = TermsId{
			Tenant:   tenant,
			Owner:    tokens[0],
			Name:     tokens[1],
			Revision: termRevision,
		}
	default:
		return nil, errors.Errorf("unknown term id format %q", s)
	}
	if err := term.Validate(); err != nil {
		return nil, errors.Trace(err)
	}
	return &term, nil
}

// MustParseTerm acts like ParseTerm but panics on error.
func MustParseTerm(s string) *TermsId {
	term, err := ParseTerm(s)
	if err != nil {
		panic(err)
	}
	return term
}

// ReadMeta reads the content of a metadata.yaml file and returns
// its representation.
func ReadMeta(r io.Reader) (*Meta, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var meta Meta
	err = yaml.Unmarshal(data, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (meta *Meta) UnmarshalYAML(f func(interface{}) error) error {
	raw := make(map[interface{}]interface{})
	err := f(&raw)
	if err != nil {
		return err
	}
	v, err := charmSchema.Coerce(raw, nil)
	if err != nil {
		return errors.New("metadata: " + err.Error())
	}

	m := v.(map[string]interface{})
	meta1, err := parseMeta(m)
	if err != nil {
		return err
	}

	if err := meta1.Check(); err != nil {
		return err
	}

	*meta = *meta1
	return nil
}

func parseMeta(m map[string]interface{}) (*Meta, error) {
	var meta Meta
	var err error

	meta.Name = m["name"].(string)
	// Schema decodes as int64, but the int range should be good
	// enough for revisions.
	meta.Summary = m["summary"].(string)
	meta.Description = m["description"].(string)
	meta.Provides = parseRelations(m["provides"], RoleProvider)
	meta.Requires = parseRelations(m["requires"], RoleRequirer)
	meta.Peers = parseRelations(m["peers"], RolePeer)
	if meta.ExtraBindings, err = parseMetaExtraBindings(m["extra-bindings"]); err != nil {
		return nil, err
	}
	meta.Categories = parseStringList(m["categories"])
	meta.Tags = parseStringList(m["tags"])
	if subordinate := m["subordinate"]; subordinate != nil {
		meta.Subordinate = subordinate.(bool)
	}
	meta.Series = parseStringList(m["series"])
	meta.Storage = parseStorage(m["storage"])
	meta.Devices = parseDevices(m["devices"])
	meta.Deployment, err = parseDeployment(m["deployment"], meta.Series, meta.Storage)
	if err != nil {
		return nil, err
	}
	meta.PayloadClasses = parsePayloadClasses(m["payloads"])

	if ver := m["min-juju-version"]; ver != nil {
		minver, err := version.Parse(ver.(string))
		if err != nil {
			return &meta, errors.Annotate(err, "invalid min-juju-version")
		}
		meta.MinJujuVersion = minver
	}
	meta.Terms = parseStringList(m["terms"])

	meta.Resources, err = parseMetaResources(m["resources"])
	if err != nil {
		return nil, err
	}

	// v2 parsing
	meta.Systems, err = parseSystems(m["systems"], meta.Resources, true)
	if err != nil {
		return nil, errors.Annotatef(err, "parsing systems")
	}
	meta.Platforms, err = parsePlatforms(m["platforms"])
	if err != nil {
		return nil, errors.Annotatef(err, "parsing platforms")
	}
	meta.Architectures, err = parseArchitectures(m["architectures"])
	if err != nil {
		return nil, errors.Annotatef(err, "parsing architectures")
	}
	meta.Containers, err = parseContainers(m["containers"], meta.Resources, meta.Platforms, meta.Storage)
	if err != nil {
		return nil, errors.Annotatef(err, "parsing containers")
	}
	return &meta, nil
}

// MarshalYAML implements yaml.Marshaler (yaml.v2).
func (m Meta) MarshalYAML() (interface{}, error) {
	var minver string
	if m.MinJujuVersion != version.Zero {
		minver = m.MinJujuVersion.String()
	}

	return struct {
		Name           string                           `yaml:"name"`
		Summary        string                           `yaml:"summary"`
		Description    string                           `yaml:"description"`
		Provides       map[string]marshaledRelation     `yaml:"provides,omitempty"`
		Requires       map[string]marshaledRelation     `yaml:"requires,omitempty"`
		Peers          map[string]marshaledRelation     `yaml:"peers,omitempty"`
		ExtraBindings  map[string]interface{}           `yaml:"extra-bindings,omitempty"`
		Categories     []string                         `yaml:"categories,omitempty"`
		Tags           []string                         `yaml:"tags,omitempty"`
		Subordinate    bool                             `yaml:"subordinate,omitempty"`
		Series         []string                         `yaml:"series,omitempty"`
		Storage        map[string]Storage               `yaml:"storage,omitempty"`
		Devices        map[string]Device                `yaml:"devices,omitempty"`
		Deployment     *Deployment                      `yaml:"deployment,omitempty"`
		Terms          []string                         `yaml:"terms,omitempty"`
		MinJujuVersion string                           `yaml:"min-juju-version,omitempty"`
		Resources      map[string]marshaledResourceMeta `yaml:"resources,omitempty"`
		Systems        []marshaledSystem                `yaml:"systems,omitempty"`
		Platforms      []Platform                       `yaml:"platforms,omitempty"`
		Architectures  []Architecture                   `yaml:"architectures,omitempty"`
		Containers     map[string]marshaledContainer    `yaml:"containers,omitempty"`
	}{
		Name:           m.Name,
		Summary:        m.Summary,
		Description:    m.Description,
		Provides:       marshaledRelations(m.Provides),
		Requires:       marshaledRelations(m.Requires),
		Peers:          marshaledRelations(m.Peers),
		ExtraBindings:  marshaledExtraBindings(m.ExtraBindings),
		Categories:     m.Categories,
		Tags:           m.Tags,
		Subordinate:    m.Subordinate,
		Series:         m.Series,
		Storage:        m.Storage,
		Devices:        m.Devices,
		Deployment:     m.Deployment,
		Terms:          m.Terms,
		MinJujuVersion: minver,
		Resources:      marshaledResources(m.Resources),
		Systems:        marshaledSystems(m.Systems),
		Platforms:      m.Platforms,
		Architectures:  m.Architectures,
		Containers:     marshaledContainers(m.Containers),
	}, nil
}

type marshaledResourceMeta struct {
	Path        string `yaml:"filename"` // TODO(ericsnow) Change to "path"?
	Type        string `yaml:"type,omitempty"`
	Description string `yaml:"description,omitempty"`
}

func marshaledResources(rs map[string]resource.Meta) map[string]marshaledResourceMeta {
	rs1 := make(map[string]marshaledResourceMeta, len(rs))
	for name, r := range rs {
		r1 := marshaledResourceMeta{
			Path:        r.Path,
			Description: r.Description,
		}
		if r.Type != resource.TypeFile {
			r1.Type = r.Type.String()
		}
		rs1[name] = r1
	}
	return rs1
}

func marshaledRelations(relations map[string]Relation) map[string]marshaledRelation {
	marshaled := make(map[string]marshaledRelation)
	for name, relation := range relations {
		marshaled[name] = marshaledRelation(relation)
	}
	return marshaled
}

type marshaledRelation Relation

func (r marshaledRelation) MarshalYAML() (interface{}, error) {
	// See calls to ifaceExpander in charmSchema.
	var noLimit int
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

func marshaledExtraBindings(bindings map[string]ExtraBinding) map[string]interface{} {
	marshaled := make(map[string]interface{})
	for _, binding := range bindings {
		marshaled[binding.Name] = nil
	}
	return marshaled
}

type marshaledSystem systems.System

func marshaledSystems(s []systems.System) []marshaledSystem {
	marshaled := []marshaledSystem(nil)
	for _, v := range s {
		marshaled = append(marshaled, marshaledSystem(v))
	}
	return marshaled
}

func (s marshaledSystem) MarshalYAML() (interface{}, error) {
	ms := struct {
		OS       string `yaml:"os,omitempty"`
		Channel  string `yaml:"channel,omitempty"`
		Resource string `yaml:"resource,omitempty"`
	}{
		OS:       s.OS,
		Channel:  s.Channel.String(),
		Resource: s.Resource,
	}
	return ms, nil
}

type marshaledContainer Container

func marshaledContainers(c map[string]Container) map[string]marshaledContainer {
	marshaled := make(map[string]marshaledContainer)
	for k, v := range c {
		marshaled[k] = marshaledContainer(v)
	}
	return marshaled
}

func (c marshaledContainer) MarshalYAML() (interface{}, error) {
	mc := struct {
		Systems []marshaledSystem `yaml:"systems,omitempty"`
		Mounts  []Mount           `yaml:"mounts,omitempty"`
	}{
		Systems: marshaledSystems(c.Systems),
		Mounts:  c.Mounts,
	}
	return mc, nil
}

// Check checks that the metadata is well-formed.
func (m Meta) Check() error {
	// Check for duplicate or forbidden relation names or interfaces.
	names := make(map[string]bool)
	checkRelations := func(src map[string]Relation, role RelationRole) error {
		for name, rel := range src {
			if rel.Name != name {
				return errors.Errorf("charm %q has mismatched relation name %q; expected %q", m.Name, rel.Name, name)
			}
			if rel.Role != role {
				return errors.Errorf("charm %q has mismatched role %q; expected %q", m.Name, rel.Role, role)
			}
			// Container-scoped require relations on subordinates are allowed
			// to use the otherwise-reserved juju-* namespace.
			if !m.Subordinate || role != RoleRequirer || rel.Scope != ScopeContainer {
				if reserved, _ := reservedName(m.Name, name); reserved {
					return errors.Errorf("charm %q using a reserved relation name: %q", m.Name, name)
				}
			}
			if role != RoleRequirer {
				if reserved, _ := reservedName(m.Name, rel.Interface); reserved {
					return errors.Errorf("charm %q relation %q using a reserved interface: %q", m.Name, name, rel.Interface)
				}
			}
			if names[name] {
				return errors.Errorf("charm %q using a duplicated relation name: %q", m.Name, name)
			}
			names[name] = true
		}
		return nil
	}
	if err := checkRelations(m.Provides, RoleProvider); err != nil {
		return err
	}
	if err := checkRelations(m.Requires, RoleRequirer); err != nil {
		return err
	}
	if err := checkRelations(m.Peers, RolePeer); err != nil {
		return err
	}

	if err := validateMetaExtraBindings(m); err != nil {
		return errors.Errorf("charm %q has invalid extra bindings: %v", m.Name, err)
	}

	// Subordinate charms must have at least one relation that
	// has container scope, otherwise they can't relate to the
	// principal.
	if m.Subordinate {
		valid := false
		if m.Requires != nil {
			for _, relationData := range m.Requires {
				if relationData.Scope == ScopeContainer {
					valid = true
					break
				}
			}
		}
		if !valid {
			return errors.Errorf("subordinate charm %q lacks \"requires\" relation with container scope", m.Name)
		}
	}

	if m.Format() == FormatV1 {
		for _, series := range m.Series {
			if !IsValidSeries(series) {
				return errors.Errorf("charm %q declares invalid series: %q", m.Name, series)
			}
		}
	} else {
		// Version 2 of the metadata should not delcare a series.
		if len(m.Series) > 0 {
			// TODO (stickupkid): This will be replaced with bases in the
			// future.
			return errors.Errorf("charm %q declares both series and systems", m.Name)
		}
	}

	names = make(map[string]bool)
	for name, store := range m.Storage {
		if store.Location != "" && store.Type != StorageFilesystem {
			return errors.Errorf(`charm %q storage %q: location may not be specified for "type: %s"`, m.Name, name, store.Type)
		}
		if store.Type == "" {
			return errors.Errorf("charm %q storage %q: type must be specified", m.Name, name)
		}
		if store.CountMin < 0 {
			return errors.Errorf("charm %q storage %q: invalid minimum count %d", m.Name, name, store.CountMin)
		}
		if store.CountMax == 0 || store.CountMax < -1 {
			return errors.Errorf("charm %q storage %q: invalid maximum count %d", m.Name, name, store.CountMax)
		}
		if names[name] {
			return errors.Errorf("charm %q storage %q: duplicated storage name", m.Name, name)
		}
		names[name] = true
	}

	names = make(map[string]bool)
	for name, device := range m.Devices {
		if device.Type == "" {
			return errors.Errorf("charm %q device %q: type must be specified", m.Name, name)
		}
		if device.CountMax >= 0 && device.CountMin >= 0 && device.CountMin > device.CountMax {
			return errors.Errorf(
				"charm %q device %q: maximum count %d can not be smaller than minimum count %d",
				m.Name, name, device.CountMax, device.CountMin)
		}
		if names[name] {
			return errors.Errorf("charm %q device %q: duplicated device name", m.Name, name)
		}
		names[name] = true
	}

	for name, payloadClass := range m.PayloadClasses {
		if payloadClass.Name != name {
			return errors.Errorf("mismatch on payload class name (%q != %q)", payloadClass.Name, name)
		}
		if err := payloadClass.Validate(); err != nil {
			return err
		}
	}

	if err := validateMetaResources(m.Resources); err != nil {
		return err
	}

	for _, term := range m.Terms {
		if _, terr := ParseTerm(term); terr != nil {
			return errors.Trace(terr)
		}
	}

	return nil
}

func reservedName(charmName, endpointName string) (reserved bool, reason string) {
	if strings.HasPrefix(charmName, "juju-") {
		return false, ""
	}
	if endpointName == "juju" {
		return true, `"juju" is a reserved name`
	}
	if strings.HasPrefix(endpointName, "juju-") {
		return true, `the "juju-" prefix is reserved`
	}
	return false, ""
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

// CombinedRelations returns all defined relations, regardless of their type in
// a single map.
func (m Meta) CombinedRelations() map[string]Relation {
	combined := make(map[string]Relation)
	for name, relation := range m.Provides {
		combined[name] = relation
	}
	for name, relation := range m.Requires {
		combined[name] = relation
	}
	for name, relation := range m.Peers {
		combined[name] = relation
	}
	return combined
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

func parseDevices(devices interface{}) map[string]Device {
	if devices == nil {
		return nil
	}
	result := make(map[string]Device)
	for name, device := range devices.(map[string]interface{}) {
		deviceMap := device.(map[string]interface{})
		device := Device{
			Name:     name,
			Type:     DeviceType(deviceMap["type"].(string)),
			CountMin: 1,
			CountMax: 1,
		}
		if desc, ok := deviceMap["description"].(string); ok {
			device.Description = desc
		}
		if countmin, ok := deviceMap["countmin"].(int64); ok {
			device.CountMin = countmin
		}
		if countmax, ok := deviceMap["countmax"].(int64); ok {
			device.CountMax = countmax
		}
		result[name] = device
	}
	return result
}

func parseDeployment(deployment interface{}, charmSeries []string, storage map[string]Storage) (*Deployment, error) {
	if deployment == nil {
		return nil, nil
	}
	if len(charmSeries) == 0 {
		return nil, errors.New("charm with deployment metadata must declare at least one series")
	}
	if charmSeries[0] != kubernetes {
		return nil, errors.Errorf("charms with deployment metadata only supported for %q", kubernetes)
	}
	deploymentMap := deployment.(map[string]interface{})
	var result Deployment
	if deploymentType, ok := deploymentMap["type"].(string); ok {
		result.DeploymentType = DeploymentType(deploymentType)
	}
	if deploymentMode, ok := deploymentMap["mode"].(string); ok {
		result.DeploymentMode = DeploymentMode(deploymentMode)
	}
	if serviceType, ok := deploymentMap["service"].(string); ok {
		result.ServiceType = ServiceType(serviceType)
	}
	if minVersion, ok := deploymentMap["min-version"].(string); ok {
		result.MinVersion = minVersion
	}
	if result.ServiceType != "" {
		osForSeries, err := series.GetOSFromSeries(charmSeries[0])
		if err != nil {
			return nil, errors.NotValidf("series %q", charmSeries[0])
		}
		valid := false
		allowed := validServiceTypes[osForSeries]
		for _, st := range allowed {
			if st == result.ServiceType {
				valid = true
				break
			}
		}
		if !valid {
			return nil, errors.NotValidf("service type %q for OS %q", result.ServiceType, osForSeries)
		}
	}
	return &result, nil
}

func parseSystems(input interface{}, resources map[string]resource.Meta, disallowResource bool) ([]systems.System, error) {
	var err error
	if input == nil {
		return nil, nil
	}
	res := []systems.System(nil)
	for _, v := range input.([]interface{}) {
		system := systems.System{}
		systemMap := v.(map[string]interface{})
		if value, ok := systemMap["os"]; ok {
			system.OS = value.(string)
		}
		if value, ok := systemMap["channel"]; ok {
			system.Channel, err = channel.Parse(value.(string))
			if err != nil {
				return nil, errors.Annotatef(err, "parsing channel %q", value.(string))
			}
		}
		if value, ok := systemMap["resource"]; ok {
			system.Resource = value.(string)
		}
		err = system.Validate()
		if err != nil {
			return nil, errors.Trace(err)
		}
		if system.Resource != "" && disallowResource {
			return nil, errors.Errorf("resource not supported as charm base system")
		}
		if system.Resource != "" {
			if r, ok := resources[system.Resource]; !ok {
				return nil, errors.NotFoundf("referenced resource %q", system.Resource)
			} else if r.Type != resource.TypeContainerImage {
				return nil, errors.Errorf("referenced resource %q is not a %s",
					system.Resource,
					resource.TypeContainerImage.String())
			}
		}
		res = append(res, system)
	}
	return res, nil
}

func parsePlatforms(input interface{}) ([]Platform, error) {
	if input == nil {
		return nil, nil
	}
	platforms := []Platform(nil)
	for _, v := range input.([]interface{}) {
		platforms = append(platforms, Platform(v.(string)))
	}
	return platforms, nil
}

func parseArchitectures(input interface{}) ([]Architecture, error) {
	if input == nil {
		return nil, nil
	}
	architectures := []Architecture(nil)
	for _, v := range input.([]interface{}) {
		architectures = append(architectures, Architecture(v.(string)))
	}
	return architectures, nil
}

func parseContainers(input interface{}, resources map[string]resource.Meta, platforms []Platform, storage map[string]Storage) (map[string]Container, error) {
	var err error
	if input == nil {
		return nil, nil
	}
	if len(platforms) != 1 || platforms[0] != PlatformKubernetes {
		return nil, errors.Errorf("containers are currently only supported on kubernetes platform")
	}
	containers := map[string]Container{}
	for name, v := range input.(map[string]interface{}) {
		containerMap := v.(map[string]interface{})
		container := Container{}
		container.Systems, err = parseSystems(containerMap["systems"], resources, false)
		if err != nil {
			return nil, errors.Annotatef(err, "container %q", name)
		}
		container.Mounts, err = parseMounts(containerMap["mounts"], storage)
		if err != nil {
			return nil, errors.Annotatef(err, "container %q", name)
		}
		containers[name] = container
	}
	if len(containers) == 0 {
		return nil, nil
	}
	return containers, nil
}

func parseMounts(input interface{}, storage map[string]Storage) ([]Mount, error) {
	if input == nil {
		return nil, nil
	}
	mounts := []Mount(nil)
	for _, v := range input.([]interface{}) {
		mount := Mount{}
		mountMap := v.(map[string]interface{})
		if value, ok := mountMap["storage"].(string); ok {
			mount.Storage = value
		}
		if value, ok := mountMap["location"].(string); ok {
			mount.Location = value
		}
		if mount.Storage == "" {
			return nil, errors.Errorf("storage must be specifed on mount")
		}
		if mount.Location == "" {
			return nil, errors.Errorf("location must be specifed on mount")
		}
		if _, ok := storage[mount.Storage]; !ok {
			return nil, errors.NotValidf("storage %q", mount.Storage)
		}
		mounts = append(mounts, mount)
	}
	return mounts, nil
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

var deviceSchema = schema.FieldMap(
	schema.Fields{
		"description": schema.String(),
		"type":        schema.String(),
		"countmin":    deviceCountC{},
		"countmax":    deviceCountC{},
	}, schema.Defaults{
		"description": schema.Omit,
		"countmin":    schema.Omit,
		"countmax":    schema.Omit,
	},
)

type deviceCountC struct{}

func (c deviceCountC) Coerce(v interface{}, path []string) (interface{}, error) {
	s, err := schema.Int().Coerce(v, path)
	if err != nil {
		return 0, err
	}
	if m, ok := s.(int64); ok {
		if m >= 0 {
			return m, nil
		}
	}
	return 0, errors.Errorf("invalid device count %d", s)
}

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
			return nil, errors.Errorf("%s: invalid count %v", strings.Join(path[1:], ""), m)
		}
		return [2]int{int(m), int(m)}, nil
	}
	match := storageCountRE.FindStringSubmatch(s.(string))
	if match == nil {
		return nil, errors.Errorf("%s: value %q does not match 'm', 'm-n', or 'm+'", strings.Join(path[1:], ""), s)
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

var deploymentSchema = schema.FieldMap(
	schema.Fields{
		"type": schema.OneOf(
			schema.Const(string(DeploymentStateful)),
			schema.Const(string(DeploymentStateless)),
			schema.Const(string(DeploymentDaemon)),
		),
		"mode": schema.OneOf(
			schema.Const(string(ModeOperator)),
			schema.Const(string(ModeWorkload)),
		),
		"service": schema.OneOf(
			schema.Const(string(ServiceCluster)),
			schema.Const(string(ServiceLoadBalancer)),
			schema.Const(string(ServiceExternal)),
			schema.Const(string(ServiceOmit)),
		),
		"min-version": schema.String(),
	}, schema.Defaults{
		"type":        schema.Omit,
		"mode":        string(ModeWorkload),
		"service":     schema.Omit,
		"min-version": schema.Omit,
	},
)

var systemSchema = schema.FieldMap(
	schema.Fields{
		"os": schema.OneOf(
			schema.Const(systems.Ubuntu),
			schema.Const(systems.Windows),
			schema.Const(systems.CentOS),
			schema.Const(systems.OpenSUSE),
			schema.Const(systems.GenericLinux),
			schema.Const(systems.OSX),
		),
		"channel":  schema.String(),
		"resource": schema.String(),
	}, schema.Defaults{
		"os":       schema.Omit,
		"channel":  schema.Omit,
		"resource": schema.Omit,
	})

var containerSchema = schema.FieldMap(
	schema.Fields{
		"systems": schema.List(systemSchema),
		"mounts":  schema.List(mountSchema),
	}, schema.Defaults{
		"systems": schema.Omit,
		"mounts":  schema.Omit,
	})

var mountSchema = schema.FieldMap(
	schema.Fields{
		"storage":  schema.String(),
		"location": schema.String(),
	}, schema.Defaults{
		"storage":  schema.Omit,
		"location": schema.Omit,
	})

var charmSchema = schema.FieldMap(
	schema.Fields{
		"name":             schema.String(),
		"summary":          schema.String(),
		"description":      schema.String(),
		"peers":            schema.StringMap(ifaceExpander(nil)),
		"provides":         schema.StringMap(ifaceExpander(nil)),
		"requires":         schema.StringMap(ifaceExpander(nil)),
		"extra-bindings":   extraBindingsSchema,
		"revision":         schema.Int(), // Obsolete
		"format":           schema.Int(), // Obsolete
		"subordinate":      schema.Bool(),
		"categories":       schema.List(schema.String()),
		"tags":             schema.List(schema.String()),
		"series":           schema.List(schema.String()),
		"storage":          schema.StringMap(storageSchema),
		"devices":          schema.StringMap(deviceSchema),
		"deployment":       deploymentSchema,
		"payloads":         schema.StringMap(payloadClassSchema),
		"resources":        schema.StringMap(resourceSchema),
		"terms":            schema.List(schema.String()),
		"min-juju-version": schema.String(),
		"platforms":        schema.List(schema.String()),
		"architectures":    schema.List(schema.String()),
		"systems":          schema.List(systemSchema),
		"containers":       schema.StringMap(containerSchema),
	},
	schema.Defaults{
		"provides":         schema.Omit,
		"requires":         schema.Omit,
		"peers":            schema.Omit,
		"extra-bindings":   schema.Omit,
		"revision":         schema.Omit,
		"format":           schema.Omit,
		"subordinate":      schema.Omit,
		"categories":       schema.Omit,
		"tags":             schema.Omit,
		"series":           schema.Omit,
		"storage":          schema.Omit,
		"devices":          schema.Omit,
		"deployment":       schema.Omit,
		"payloads":         schema.Omit,
		"resources":        schema.Omit,
		"terms":            schema.Omit,
		"min-juju-version": schema.Omit,
		"platforms":        schema.Omit,
		"architectures":    schema.Omit,
		"systems":          schema.Omit,
		"containers":       schema.Omit,
	},
)
