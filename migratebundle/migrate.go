package migratebundle

import (
	"github.com/juju/errgo"
	"gopkg.in/yaml.v2"
)

// oldBundle represents a legacy bundle.
type oldBundle struct {
	Series   string      `yaml:",omitempty"`
	Inherits interface{} `yaml:",omitempty"` // string or []string
	Services map[string]*oldService
	// A relation can be in one of two styles:
	// ["r1", "r2"] or ["r1", ["r2", "r3", ...]]
	Relations []interface{}          `yaml:",omitempty"` // []string or []interface{}{"", []string{...}}
	Overrides map[string]interface{} `yaml:",omitempty"`
}

// oldBundle represents a service from a legacy bundle.
type oldService struct {
	Charm       string                 `yaml:",omitempty"`
	Branch      string                 `yaml:",omitempty"`
	NumUnits    *int                   `yaml:"num_units,omitempty"`
	Constraints string                 `yaml:",omitempty"`
	Expose      bool                   `yaml:",omitempty"`
	Annotations map[string]interface{} `yaml:",omitempty"`

	// To can be a string or an integer.
	To interface{} `yaml:",omitempty"`

	Options map[string]interface{} `yaml:",omitempty"`

	// Spurious fields, used by existing bundles but not
	// valid in the specification. Kept here so that
	// the reversability tests can work.
	Name    string `yaml:",omitempty"`
	Exposed bool   `yaml:",omitempty"`
	Local   string `yaml:",omitempty"`
}

// TODO (this is where we're headed)
// Migrate parses the old-style bundles.yaml file in bundlesYAML
// and returns a map containing an entry for each bundle
// found in that basket, keyed by the name of the bundle.
//func Migrate(bundlesYAML []byte) (map[string] charm.Bundle, error)

// inherit adds any inherited attributes to the given bundle b. It does
// not modify b, returning a new bundle if necessary.
//
// The bundles map holds all the bundles from the basket (the possible
// bundles that can be inherited from).
func inherit(b *oldBundle, bundles map[string]*oldBundle) (*oldBundle, error) {
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
	var newb oldBundle
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
