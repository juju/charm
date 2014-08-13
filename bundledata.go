// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/juju/names"
	goyaml "gopkg.in/yaml.v1"
)

// BundleData holds the contents of the bundle.
type BundleData struct {
	// Services holds one entry for each service
	// that the bundle will create, indexed by
	// the service name.
	Services map[string]*ServiceSpec

	// Machines holds one entry for each machine referred to
	// by unit placements. These will be mapped onto actual
	// machines at bundle deployment time.
	// It is an error if a machine is specified but
	// not referred to by a unit placement directive.
	Machines map[string]*MachineSpec `bson:",omitempty" json:",omitempty" yaml:",omitempty"`

	// Series holds the default series to use when
	// the bundle chooses charms.
	Series string `bson:",omitempty" json:",omitempty" yaml:",omitempty"`

	// Relations holds a slice of 2-element slices,
	// each specifying a relation between two services.
	// Each two-element slice holds two colon-separated
	// (service, relation) pairs - the relation is made between
	// each.
	Relations [][]string `bson:",omitempty" json:",omitempty" yaml:",omitempty"`

	// White listed set of tags to categorize bundles as we do charms.
	Tags []string `bson:",omitempty" json:",omitempty" yaml:",omitempty"`

	// Short paragraph explaining what the bundle is useful for.
	Description string `bson:",omitempty" json:",omitempty" yaml:",omitempty"`
}

// MachineSpec represents a notional machine that will be mapped
// onto an actual machine at bundle deployment time.
type MachineSpec struct {
	Constraints string            `bson:",omitempty" json:",omitempty" yaml:",omitempty"`
	Annotations map[string]string `bson:",omitempty" json:",omitempty" yaml:",omitempty"`
}

// ServiceSpec represents a single service that will
// be deployed as part of the bundle.
type ServiceSpec struct {
	// Charm holds the charm URL of the charm to
	// use for the given service.
	Charm string

	// NumUnits holds the number of units of the
	// service that will be deployed.
	NumUnits int `yaml:"num_units"`

	// To may hold up to NumUnits members with
	// each member specifying a desired placement
	// for the respective unit of the service.
	//
	// In regular-expression-like notation, each
	// element matches the following pattern:
	//
	//      (<containertype>:)?(<unit>|<machine>|new)
	//
	// If containertype is specified, the unit is deployed
	// into a new container of that type, otherwise
	// it will be "hulk-smashed" into the specified location,
	// by co-locating it with any other units that happen to
	// be there, which may result in unintended behavior.
	//
	// The second part (after the colon) specifies where
	// the new unit should be placed - it may refer to
	// a unit of another service specified in the bundle,
	// a machine id specified in the machines section,
	// or the special name "new" which specifies a newly
	// created machine.
	//
	// A unit placement may be specified with a service name only,
	// in which case its unit number is assumed to
	// be one more than the unit number of the previous
	// unit in the list with the same service, or zero
	// if there were none.
	//
	// If there are less elements in To than NumUnits,
	// the last element is replicated to fill it. If there
	// are no elements (or To is omitted), "new" is replicated.
	//
	// For example:
	//
	//     wordpress/0 wordpress/1 lxc:0 kvm:new
	//
	//  specifies that the first two units get hulk-smashed
	//  onto the first two units of the wordpress service,
	//  the third unit gets allocated onto an lxc container
	//  on machine 0, and subsequent units get allocated
	//  on kvm containers on new machines.
	//
	// The above example is the same as this:
	//
	//     wordpress wordpress lxc:0 kvm:new
	To []string `bson:",omitempty" json:",omitempty" yaml:",omitempty"`

	// Options holds the configuration values
	// to apply to the new service. They should
	// be compatible with the charm configuration.
	Options map[string]interface{} `bson:",omitempty" json:",omitempty" yaml:",omitempty"`

	// Annotations holds any annotations to apply to the
	// service when deployed.
	Annotations map[string]string `bson:",omitempty" json:",omitempty" yaml:",omitempty"`

	// Constraints holds the default constraints to apply
	// when creating new machines for units of the service.
	// This is ignored for units with explicit placement directives.
	Constraints string `bson:",omitempty" json:",omitempty" yaml:",omitempty"`
}

// ReadBundleData reads bundle data from the given reader.
// The returned data is not verified - call Verify to ensure
// that it is OK.
func ReadBundleData(r io.Reader) (*BundleData, error) {
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var bd BundleData
	if err := goyaml.Unmarshal(bytes, &bd); err != nil {
		return nil, fmt.Errorf("cannot unmarshal bundle data: %v", err)
	}
	return &bd, nil
}

// VerificationError holds an error generated by BundleData.Verify,
// holding all the verification errors found when verifying.
type VerificationError struct {
	Errors []error
}

func (err *VerificationError) Error() string {
	switch len(err.Errors) {
	case 0:
		return "no verification errors!"
	case 1:
		return err.Errors[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", err.Errors[0], len(err.Errors)-1)
}

type bundleDataVerifier struct {
	bd *BundleData

	// machines holds the reference counts of all machines
	// as referred to by placement directives.
	machineRefCounts map[string]int

	charms map[string]Charm

	errors            []error
	verifyConstraints func(c string) error
}

func (verifier *bundleDataVerifier) addErrorf(f string, a ...interface{}) {
	verifier.addError(fmt.Errorf(f, a...))
}

func (verifier *bundleDataVerifier) addError(err error) {
	verifier.errors = append(verifier.errors, err)
}

func (verifier *bundleDataVerifier) err() error {
	if len(verifier.errors) > 0 {
		return &VerificationError{verifier.errors}
	}
	return nil
}

// RequiredCharms returns a sorted slice of all the charm URLs
// required by the bundle.
func (bd *BundleData) RequiredCharms() []string {
	req := make([]string, 0, len(bd.Services))
	for _, svc := range bd.Services {
		req = append(req, svc.Charm)
	}
	sort.Strings(req)
	return req
}

// Verify is a convenience method that calls VerifyWithCharms
// with a nil charms map.
func (bd *BundleData) Verify(
	verifyConstraints func(c string) error,
) error {
	return bd.VerifyWithCharms(verifyConstraints, nil)
}

// VerifyWithCharms verifies that the bundle is consistent.
// The verifyConstraints function is called to verify any constraints
// that are found.
//
// It verifies the following:
//
// - All defined machines are referred to by placement directives.
// - All services referred to by placement directives are specified in the bundle.
// - All services referred to by relations are specified in the bundle.
// - All constraints are valid.
//
// If charms is not nil, it should hold a map with an entry for each
// charm url returned by bd.RequiredCharms. The verification will then
// also check that services are defined with valid charms,
// relations are correctly made and options are defined correctly.
//
// If the verification fails, Verify returns a *VerificationError describing
// all the problems found.
func (bd *BundleData) VerifyWithCharms(
	verifyConstraints func(c string) error,
	charms map[string]Charm,
) error {
	verifier := &bundleDataVerifier{
		verifyConstraints: verifyConstraints,
		bd:                bd,
		machineRefCounts:  make(map[string]int),
		charms:            charms,
	}
	for id := range bd.Machines {
		verifier.machineRefCounts[id] = 0
	}
	if bd.Series != "" && !IsValidSeries(bd.Series) {
		verifier.addErrorf("bundle declares an invalid series %q", bd.Series)
	}
	verifier.verifyMachines()
	verifier.verifyServices()
	verifier.verifyRelations()
	verifier.verifyOptions()

	for id, count := range verifier.machineRefCounts {
		if count == 0 {
			verifier.addErrorf("machine %q is not referred to by a placement directive", id)
		}
	}
	return verifier.err()
}

var validMachineId = regexp.MustCompile("^" + names.NumberSnippet + "$")

func (verifier *bundleDataVerifier) verifyMachines() {
	for id, m := range verifier.bd.Machines {
		if !validMachineId.MatchString(id) {
			verifier.addErrorf("invalid machine id %q found in machines", id)
		}
		if m == nil {
			continue
		}
		if m.Constraints != "" {
			if err := verifier.verifyConstraints(m.Constraints); err != nil {
				verifier.addErrorf("invalid constraints %q in machine %q: %v", m.Constraints, id, err)
			}
		}
	}
}

func (verifier *bundleDataVerifier) verifyServices() {
	if len(verifier.bd.Services) == 0 {
		verifier.addErrorf("at least one service must be specified")
		return
	}
	for name, svc := range verifier.bd.Services {
		if _, err := ParseReference(svc.Charm); err != nil {
			verifier.addErrorf("invalid charm URL in service %q: %v", name, err)
		}
		if err := verifier.verifyConstraints(svc.Constraints); err != nil {
			verifier.addErrorf("invalid constraints %q in service %q: %v", svc.Constraints, name, err)
		}
		verifier.verifyPlacement(svc.To)
		if svc.NumUnits < 0 {
			verifier.addErrorf("negative number of units specified on service %q", name)
		} else if len(svc.To) > svc.NumUnits {
			verifier.addErrorf("too many units specified in unit placement for service %q", name)
		}
		if verifier.charms != nil {
			if _, ok := verifier.charms[svc.Charm]; !ok {
				verifier.addErrorf("service %q refers to non-existent charm %q", name, svc.Charm)
			}
		}
	}
}

func (verifier *bundleDataVerifier) verifyPlacement(to []string) {
	for _, p := range to {
		up, err := ParsePlacement(p)
		if err != nil {
			verifier.addError(err)
			continue
		}
		switch {
		case up.Service != "":
			spec, ok := verifier.bd.Services[up.Service]
			if !ok {
				verifier.addErrorf("placement %q refers to a service not defined in this bundle", p)
				continue
			}
			if up.Unit >= 0 && up.Unit >= spec.NumUnits {
				verifier.addErrorf("placement %q specifies a unit greater than the %d unit(s) started by the target service", p, spec.NumUnits)
			}
		case up.Machine == "new":
		default:
			_, ok := verifier.bd.Machines[up.Machine]
			if !ok {
				verifier.addErrorf("placement %q refers to a machine not defined in this bundle", p)
				continue
			}
			verifier.machineRefCounts[up.Machine]++
		}
	}
}

func (verifier *bundleDataVerifier) verifyRelations() {
	seen := make(map[[2]endpoint]bool)
	for _, relPair := range verifier.bd.Relations {
		if len(relPair) != 2 {
			verifier.addErrorf("relation %q has %d endpoint(s), not 2", relPair, len(relPair))
			continue
		}
		var epPair [2]endpoint
		relParseErr := false
		for i, svcRel := range relPair {
			ep, err := parseEndpoint(svcRel)
			if err != nil {
				verifier.addError(err)
				relParseErr = true
				continue
			}
			if _, ok := verifier.bd.Services[ep.service]; !ok {
				verifier.addErrorf("relation %q refers to service %q not defined in this bundle", relPair, ep.service)
			}
			epPair[i] = ep
		}
		if relParseErr {
			// We failed to parse at least one relation, so don't
			// bother checking further.
			continue
		}
		if epPair[0].service == epPair[1].service {
			verifier.addErrorf("relation %q relates a service to itself", relPair)
		}
		// Re-order pairs so that we diagnose duplicate relations
		// whichever way they're specified.
		if epPair[1].less(epPair[0]) {
			epPair[1], epPair[0] = epPair[0], epPair[1]
		}
		if _, ok := seen[epPair]; ok {
			verifier.addErrorf("relation %q is defined more than once", relPair)
		}
		verifier.verifyRelation(epPair[0], epPair[1])
		seen[epPair] = true
	}
}

// verifyRelation verifies a single relation.
// It checks that both endpoints of the relation are
// defined, and that the relationship is correctly
// symmetrical (provider to requirer) and shares
// the same interface.
func (verifier *bundleDataVerifier) verifyRelation(ep0, ep1 endpoint) {
	if verifier.charms == nil {
		// No charms to verify against.
		return
	}
	svc0 := verifier.bd.Services[ep0.service]
	svc1 := verifier.bd.Services[ep1.service]
	if svc0 == nil || svc1 == nil || svc0 == svc1 {
		// An error will be produced by verifyRelations for this case.
		return
	}
	charm0 := verifier.charms[svc0.Charm]
	charm1 := verifier.charms[svc1.Charm]
	if charm0 == nil || charm1 == nil {
		// An error will be produced by verifyServices for this case.
		return
	}
	relProv0, okProv0 := charm0.Meta().Provides[ep0.relation]
	relReq0, okReq0 := charm0.Meta().Requires[ep0.relation]
	if !okProv0 && !okReq0 {
		verifier.addErrorf("charm %q used by service %q does not define relation %q", svc0.Charm, ep0.service, ep0.relation)
	}
	relProv1, okProv1 := charm1.Meta().Provides[ep1.relation]
	relReq1, okReq1 := charm1.Meta().Requires[ep1.relation]
	if !okProv1 && !okReq1 {
		verifier.addErrorf("charm %q used by service %q does not define relation %q", svc1.Charm, ep1.service, ep1.relation)
	}

	var relProv, relReq Relation
	var epProv, epReq endpoint
	switch {
	case okProv0 && okReq1:
		relProv, relReq = relProv0, relReq1
		epProv, epReq = ep0, ep1
	case okReq0 && okProv1:
		relProv, relReq = relProv1, relReq0
		epProv, epReq = ep1, ep0
	case okProv0 && okProv1:
		verifier.addErrorf("relation %q to %q relates provider to provider", ep0, ep1)
		return
	case okReq0 && okReq1:
		verifier.addErrorf("relation %q to %q relates requirer to requirer", ep0, ep1)
		return
	default:
		// Errors were added above.
		return
	}
	if relProv.Interface != relReq.Interface {
		verifier.addErrorf("mismatched interface between %q and %q (%q vs %q)", epProv, epReq, relProv.Interface, relReq.Interface)
	}
}

// verifyOptions verifies that the options are correctly defined
// with respect to the charm config options.
func (verifier *bundleDataVerifier) verifyOptions() {
	if verifier.charms == nil {
		return
	}
	for svcName, svc := range verifier.bd.Services {
		charm := verifier.charms[svc.Charm]
		if charm == nil {
			// An error will be produced by verifyServices for this case.
			continue
		}
		config := charm.Config()
		for name, value := range svc.Options {
			opt, ok := config.Options[name]
			if !ok {
				verifier.addErrorf("cannot validate service %q: configuration option %q not found in charm %q", svcName, name, svc.Charm)
				continue
			}
			_, err := opt.validate(name, value)
			if err != nil {
				verifier.addErrorf("cannot validate service %q: %v", svcName, err)
			}
		}
	}
}

var validServiceRelation = regexp.MustCompile("^(" + names.ServiceSnippet + "):(" + names.RelationSnippet + ")$")

type endpoint struct {
	service  string
	relation string
}

func (ep endpoint) String() string {
	return fmt.Sprintf("%s:%s", ep.service, ep.relation)
}

func (ep1 endpoint) less(ep2 endpoint) bool {
	if ep1.service == ep2.service {
		return ep1.relation < ep2.relation
	}
	return ep1.service < ep2.service
}

func parseEndpoint(ep string) (endpoint, error) {
	m := validServiceRelation.FindStringSubmatch(ep)
	if m == nil {
		return endpoint{}, fmt.Errorf("invalid relation syntax %q", ep)
	}
	return endpoint{
		service:  m[1],
		relation: m[2],
	}, nil
}

type UnitPlacement struct {
	// ContainerType holds the container type of the new
	// new unit, or empty if unspecified.
	ContainerType string

	// Machine holds the numeric machine id, or "new",
	// or empty if the placement specifies a service.
	Machine string

	// Service holds the service name, or empty if
	// the placement specifies a machine.
	Service string

	// Unit holds the unit number of the service, or -1
	// if unspecified.
	Unit int
}

var snippetReplacer = strings.NewReplacer(
	"container", names.ContainerTypeSnippet,
	"number", names.NumberSnippet,
	"service", names.ServiceSnippet,
)

// validPlacement holds regexp that matches valid placement requests. To
// make the expression easier to comprehend and maintain, we replace
// symbolic snippet references in the regexp by their actual regexps
// using snippetReplacer.
var validPlacement = regexp.MustCompile(
	snippetReplacer.Replace(
		"^(?:(container):)?(?:(service)(?:/(number))?|(number))$",
	),
)

// ParsePlacement parses a unit placement directive, as
// specified in the To clause of a service entry in the
// services section of a bundle.
func ParsePlacement(p string) (*UnitPlacement, error) {
	m := validPlacement.FindStringSubmatch(p)
	if m == nil {
		return nil, fmt.Errorf("invalid placement syntax %q", p)
	}
	up := UnitPlacement{
		ContainerType: m[1],
		Service:       m[2],
		Machine:       m[4],
	}
	if unitStr := m[3]; unitStr != "" {
		// We know that unitStr must be a valid integer because
		// it's specified as such in the regexp.
		up.Unit, _ = strconv.Atoi(unitStr)
	} else {
		up.Unit = -1
	}
	if up.Service == "new" {
		if up.Unit != -1 {
			return nil, fmt.Errorf("invalid placement syntax %q", p)
		}
		up.Machine, up.Service = "new", ""
	}
	return &up, nil
}
