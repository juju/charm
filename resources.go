// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"

	"github.com/juju/schema"

	"gopkg.in/juju/charm.v6-unstable/resource"
)

var resourceSchema = schema.FieldMap(
	schema.Fields{
		"type":     schema.String(),
		"filename": schema.String(), // TODO(ericsnow) Change to "path"?
		"comment":  schema.String(),
	},
	schema.Defaults{
		"type":    resource.ResourceTypeFile.String(),
		"comment": "",
	},
)

func parseResources(data interface{}) map[string]resource.Resource {
	if data == nil {
		return nil
	}

	result := make(map[string]resource.Resource)
	for name, val := range data.(map[string]interface{}) {
		result[name] = resource.ParseResource(name, val)
	}

	return result
}

func validateResources(resources map[string]resource.Resource) error {
	for name, resource := range resources {
		if resource.Name != name {
			return fmt.Errorf("mismatch on resource name (%q != %q)", resource.Name, name)
		}
		if err := resource.Validate(); err != nil {
			return err
		}
	}
	return nil
}
