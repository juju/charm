package migratebundle

import (
	"github.com/juju/errgo"
	"gopkg.in/yaml.v1"

	"gopkg.in/juju/charm.v4"
)

// legacyBundle represents an old-style bundle.
type legacyBundle struct {
	Series   string      `yaml:",omitempty"`
	Inherits interface{} `yaml:",omitempty"` // string or []string
	Services map[string]*legacyService
	// A relation can be in one of two styles:
	// ["r1", "r2"] or ["r1", ["r2", "r3", ...]]
	Relations []interface{}          `yaml:",omitempty"` // []string or []interface{}{"", []string{...}}
	Overrides map[string]interface{} `yaml:",omitempty"`
}

// legacyService represents a service from a legacy bundle.
type legacyService struct {
	Charm       string                 `yaml:",omitempty"`
	Branch      string                 `yaml:",omitempty"`
	NumUnits    *int                   `yaml:"num_units,omitempty"`
	Constraints string                 `yaml:",omitempty"`
	Expose      bool                   `yaml:",omitempty"`
	Annotations map[string]string      `yaml:",omitempty"`
	To          string                 `yaml:",omitempty"`
	Options     map[string]interface{} `yaml:",omitempty"`

	// Spurious fields, used by existing bundles but not
	// valid in the specification. Kept here so that
	// the reversability tests can work.
	Name    string `yaml:",omitempty"`
	Exposed bool   `yaml:",omitempty"`
	Local   string `yaml:",omitempty"`
}

// Migrate parses the old-style bundles.yaml file in bundlesYAML
// and returns a map containing an entry for each bundle
// found in that basket, keyed by the name of the bundle.
//
// It performs the following changes:
//
// - Any inheritance is expanded.
//
// - when a "to" placement directive refers to machine 0,
// an explicit machines section is added. Also, convert
// it to a slice.
//
// - If the charm URL is not specified, it is taken from the
// service name.
//
// - num_units is renamed to numunits, and set to 1 if omitted.
//
// - A relation clause with multiple targets is expanded
// into multiple relation clauses.
func Migrate(bundlesYAML []byte) (map[string]*charm.BundleData, error) {
	var bundles map[string]*legacyBundle
	if err := yaml.Unmarshal(bundlesYAML, &bundles); err != nil {
		return nil, errgo.Notef(err, "cannot parse legacy bundle")
	}
	// First expand any inherits clauses.
	newBundles := make(map[string]*charm.BundleData)
	for name, bundle := range bundles {
		bundle, err := inherit(bundle, bundles)
		if err != nil {
			return nil, errgo.Notef(err, "bundle inheritance failed for %q", name)
		}
		newBundle, err := migrate(bundle)
		if err != nil {
			return nil, errgo.Notef(err, "bundle migration failed for %q", name)
		}
		newBundles[name] = newBundle
	}
	return newBundles, nil
}

func migrate(b *legacyBundle) (*charm.BundleData, error) {
	data := &charm.BundleData{
		Services: make(map[string]*charm.ServiceSpec),
		Series:   b.Series,
		Machines: make(map[string]*charm.MachineSpec),
	}
	for name, svc := range b.Services {
		if svc == nil {
			svc = new(legacyService)
		}
		newSvc := &charm.ServiceSpec{
			Charm:       svc.Charm,
			NumUnits:    1, // default
			Options:     svc.Options,
			Annotations: svc.Annotations,
			Constraints: svc.Constraints,
		}
		if newSvc.Charm == "" {
			newSvc.Charm = name
		}
		if svc.NumUnits != nil {
			newSvc.NumUnits = *svc.NumUnits
		}
		if svc.To != "" {
			newSvc.To = []string{svc.To}
			place, err := charm.ParsePlacement(svc.To)
			if err != nil {
				return nil, errgo.Notef(err, "cannot parse 'to' placment clause %q", svc.To)
			}
			if place.Machine != "" {
				data.Machines[place.Machine] = new(charm.MachineSpec)
			}
		}
		data.Services[name] = newSvc
	}
	var err error
	data.Relations, err = expandRelations(b.Relations)
	if err != nil {
		return nil, errgo.Notef(err, "cannot expand relations")
	}
	if len(data.Machines) == 0 {
		data.Machines = nil
	}
	return data, nil
}

// expandRelations expands any relations that are
// in the form [r1, [r2, r3, ...]] into the form [r1, r2], [r1, r3], ....
func expandRelations(relations []interface{}) ([][]string, error) {
	var newRelations [][]string
	for _, rel := range relations {
		rel, ok := rel.([]interface{})
		if !ok || len(rel) != 2 {
			return nil, errgo.Newf("unexpected relation clause %#v", rel)
		}
		ep0, ok := rel[0].(string)
		if !ok {
			return nil, errgo.Newf("first relation endpoint is %#v not string", rel[0])
		}
		if ep1, ok := rel[1].(string); ok {
			newRelations = append(newRelations, []string{ep0, ep1})
			continue
		}
		eps, ok := rel[1].([]interface{})
		if !ok {
			return nil, errgo.Newf("second relation endpoint is %#v not list or string", rel[1])
		}
		for _, ep1 := range eps {
			ep1, ok := ep1.(string)
			if !ok {
				return nil, errgo.Newf("relation list member is not string")
			}
			newRelations = append(newRelations, []string{ep0, ep1})
		}
	}
	return newRelations, nil
}

// inherit adds any inherited attributes to the given bundle b. It does
// not modify b, returning a new bundle if necessary.
//
// The bundles map holds all the bundles from the basket (the possible
// bundles that can be inherited from).
func inherit(b *legacyBundle, bundles map[string]*legacyBundle) (*legacyBundle, error) {
	if b.Inherits == nil {
		return b, nil
	}
	// The Inherits clause can be specified as a string or a list.
	// There are no known bundles which have more than one element in
	// the list, so fail if there are, as we don't want to implement
	// multiple inheritance when we don't have to.
	inherits, ok := b.Inherits.(string)
	if !ok {
		list, ok := b.Inherits.([]interface{})
		if !ok || len(list) != 1 {
			return nil, errgo.Newf("bad inherits clause %#v", b.Inherits)
		}
		inherits, ok = list[0].(string)
		if !ok {
			return nil, errgo.Newf("bad inherits clause %#v", b.Inherits)
		}
	}
	from := bundles[inherits]
	if from == nil {
		return nil, errgo.Newf("inherited-from bundle %q not found", inherits)
	}
	if from.Inherits != nil {
		return nil, errgo.Newf("only a single level of inheritance is supported")
	}
	// Make a generic copy of both the base and target bundles,
	// so we can apply inheritance regardless of Go types.
	var target map[interface{}]interface{}
	err := yamlCopy(&target, from)
	if err != nil {
		return nil, errgo.Notef(err, "copy target")
	}
	var source map[interface{}]interface{}
	err = yamlCopy(&source, b)
	if err != nil {
		return nil, errgo.Notef(err, "copy source")
	}
	// Apply the inherited attributes.
	copyOnto(target, source, true)

	// Convert back to Go types.
	var newb legacyBundle
	err = yamlCopy(&newb, target)
	if err != nil {
		return nil, errgo.Notef(err, "copy result")
	}
	return &newb, nil
}

// yamlCopy copies the source value into the value
// pointed to by the target value by marshaling
// and unmarshaling YAML.
func yamlCopy(target, source interface{}) error {
	data, err := yaml.Marshal(source)
	if err != nil {
		return errgo.Notef(err, "marshal copy")
	}
	if err := yaml.Unmarshal(data, target); err != nil {
		return errgo.Notef(err, "unmarshal copy")
	}
	return nil
}

// copyOnto copies the source onto the target,
// preserving any of the source that is not present
// in the target.
func copyOnto(target, source map[interface{}]interface{}, isRoot bool) {
	for key, val := range source {
		if key == "inherits" && isRoot {
			continue
		}
		switch val := val.(type) {
		case map[interface{}]interface{}:
			if targetVal, ok := target[key].(map[interface{}]interface{}); ok {
				copyOnto(targetVal, val, false)
			} else {
				target[key] = val
			}
		default:
			target[key] = val
		}
	}
}
