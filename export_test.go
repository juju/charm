// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

// Export meaningful bits for tests only.

var (
	IfaceExpander = ifaceExpander
	ValidateValue = validateValue

	ParsePayloadClass = parsePayloadClass
	ResourceSchema    = resourceSchema
)

func MissingSeriesError() error {
	return missingSeriesError
}
