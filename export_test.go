// Copyright 2011-2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

// Export meaningful bits for tests only.

var (
	IfaceExpander = ifaceExpander
	ValidateValue = validateValue

	ParsePayloadClass         = parsePayloadClass
	ResourceSchema            = resourceSchema
	ExtraBindingsSchema       = extraBindingsSchema
	ValidateMetaExtraBindings = validateMetaExtraBindings
	ParseResourceMeta         = parseResourceMeta
)

func MissingSeriesError() error {
	return missingSeriesError
}
