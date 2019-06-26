// Copyright 2019 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"reflect"
	"strings"
)

// ExtractBaseAndOverlayParts splits the bundle data into a base and
// overlay-specific bundle so that their union yields bd. To decide whether a
// field is overlay-specific, the implementation uses reflection and
// recursively scans the BundleData fields looking for fields annotated with
// the "overlay-only: true" tag.
//
// To produce the base bundle, the original bundle is filtered and all
// overlay-specific values are set to the zero value for their type. To produce
// the overlay-specific bundle, we once again filter the original bundle but
// this time zero out fields that do not contain any descendant fields that are
// overlay-specific.
//
// To clarify how this method works let's consider a bundle created via the
// yaml blob below:
//
//   applications:
//     apache2:
//       charm: cs:apache2-26
//       offers:
//         my-offer:
//           endpoints:
//           - apache-website
//           - website-cache
//         my-other-offer:
//           endpoints:
//           - apache-website
//   series: bionic
//
// The "offers" and "endpoints" attributes are overlay-specific fields. If we
// were to run this method and then marshal the results back to yaml we would
// get:
//
// The base bundle:
//
//   applications:
//     apache2:
//       charm: cs:apache2-26
//   series: bionic
//
// The overlay-specific bundle:
//
//   applications:
//     apache2:
//       offers:
//         my-offer:
//           endpoints:
//           - apache-website
//           - website-cache
//         my-other-offer:
//           endpoints:
//           - apache-website
//
// The two bundles returned by this method are copies of the original bundle
// data and can thus be safely manipulated by the caller.
func ExtractBaseAndOverlayParts(bd *BundleData) (base, overlay *BundleData, err error) {
	if base, err = cloneBundleData(bd); err != nil {
		return nil, nil, err
	}

	if overlay, err = cloneBundleData(bd); err != nil {
		return nil, nil, err
	}

	_ = visitField(&visitorContext{structVisitor: clearOverlayFields}, base)
	_ = visitField(&visitorContext{structVisitor: clearNonOverlayFields}, overlay)
	return base, overlay, nil
}

// cloneBundleData uses the gob package to perform a deep copy of bd.
func cloneBundleData(bd *BundleData) (*BundleData, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(bd); err != nil {
		return nil, err
	}

	var clone *BundleData
	if err := gob.NewDecoder(&buf).Decode(&clone); err != nil {
		return nil, err
	}

	clone.unmarshaledWithServices = bd.unmarshaledWithServices
	return clone, nil
}

// VerifyNoOverlayFieldsPresent scans the contents of bd and returns an error
// if the bundle contains any overlay-specific values.
func VerifyNoOverlayFieldsPresent(bd *BundleData) error {
	var (
		errList   []error
		pathStack []string
	)

	ctx := &visitorContext{
		structVisitor: func(ctx *visitorContext, val reflect.Value, typ reflect.Type) (foundOverlay bool) {
			for i := 0; i < typ.NumField(); i++ {
				structField := typ.Field(i)

				// Skip non-exportable and empty fields
				v := val.Field(i)
				if !v.CanInterface() || isZero(v) {
					continue
				}

				if isOverlayField(structField) {
					errList = append(
						errList,
						fmt.Errorf(
							"%s.%s can only appear in an overlay section",
							strings.Join(pathStack, "."),
							yamlName(structField),
						),
					)
					foundOverlay = true
				}

				pathStack = append(pathStack, yamlName(structField))
				if visitField(ctx, v.Interface()) {
					foundOverlay = true
				}
				pathStack = pathStack[:len(pathStack)-1]
			}
			return foundOverlay
		},
		indexedElemPreVisitor: func(index interface{}) {
			pathStack = append(pathStack, fmt.Sprint(index))
		},
		indexedElemPostVisitor: func(_ interface{}) {
			pathStack = pathStack[:len(pathStack)-1]
		},
	}

	_ = visitField(ctx, bd)
	if len(errList) == 0 {
		return nil
	}

	return &VerificationError{errList}
}

func yamlName(structField reflect.StructField) string {
	fields := strings.Split(structField.Tag.Get("yaml"), ",")
	if len(fields) == 0 || fields[0] == "" {
		return strings.ToLower(structField.Name)
	}

	return fields[0]
}

type visitorContext struct {
	structVisitor func(ctx *visitorContext, val reflect.Value, typ reflect.Type) bool

	// An optional pre/post visitor for indexable items (slices, maps)
	indexedElemPreVisitor  func(index interface{})
	indexedElemPostVisitor func(index interface{})
}

// visitField invokes ctx.structVisitor(val) if v is a struct and returns back
// the visitor's result. On the other hand, if val is a slice or a map,
// visitField invoke specialized functions that support iterating such types.
func visitField(ctx *visitorContext, val interface{}) bool {
	typ := reflect.TypeOf(val)
	v := reflect.ValueOf(val)

	// De-reference pointers
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		typ = v.Type()
	}

	switch typ.Kind() {
	case reflect.Struct:
		return ctx.structVisitor(ctx, v, typ)
	case reflect.Map:
		return visitFieldsInMap(ctx, v)
	case reflect.Slice:
		return visitFieldsInSlice(ctx, v)
	}

	// v is not a struct or something we can iterate to reach a struct
	return false
}

// visitFieldsInMap iterates the map specified by val and recursively visits
// each map element. The returned value is the logical OR of the responses
// returned by visiting all map elements.
func visitFieldsInMap(ctx *visitorContext, val reflect.Value) (result bool) {
	for _, key := range val.MapKeys() {
		v := val.MapIndex(key)
		if !v.CanInterface() {
			continue
		}

		if ctx.indexedElemPreVisitor != nil {
			ctx.indexedElemPreVisitor(key)
		}

		result = visitField(ctx, v.Interface()) || result

		if ctx.indexedElemPostVisitor != nil {
			ctx.indexedElemPostVisitor(key)
		}
	}

	return result
}

// visitFieldsInSlice iterates the slice specified by val and recursively
// visits each element. The returned value is the logical OR of the responses
// returned by visiting all slice elements.
func visitFieldsInSlice(ctx *visitorContext, val reflect.Value) (result bool) {
	for i := 0; i < val.Len(); i++ {
		v := val.Index(i)
		if !v.CanInterface() {
			continue
		}

		if ctx.indexedElemPreVisitor != nil {
			ctx.indexedElemPreVisitor(i)
		}

		result = visitField(ctx, v.Interface()) || result

		if ctx.indexedElemPostVisitor != nil {
			ctx.indexedElemPostVisitor(i)
		}
	}

	return result
}

// clearOverlayFields is an implementation of structVisitor. It recursively
// visits all fields in the val struct and sets the ones that are tagged as
// overlay-only to the zero value for their particular type.
func clearOverlayFields(ctx *visitorContext, val reflect.Value, typ reflect.Type) (retainAncestors bool) {
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)

		// Skip non-exportable and empty fields
		v := val.Field(i)
		if !v.CanInterface() || isZero(v) {
			continue
		}

		// No need to recurse further down; just erase the field
		if isOverlayField(structField) {
			v.Set(reflect.Zero(v.Type()))
			continue
		}

		_ = visitField(ctx, v.Interface())
		retainAncestors = true
	}
	return retainAncestors
}

// clearNonOverlayFields is an implementation of structVisitor. It recursively
// visits all fields in the val struct and sets any field that does not contain
// any overlay-only descendants to the zero value for its particular type.
func clearNonOverlayFields(ctx *visitorContext, val reflect.Value, typ reflect.Type) (retainAncestors bool) {
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)

		// Skip non-exportable and empty fields
		v := val.Field(i)
		if !v.CanInterface() || isZero(v) {
			continue
		}

		// If this is an overlay field we need to preserve it and all
		// its ancestor fields up to the root. However, we still need
		// to visit its descendants in case we need to clear additional
		// non-overlay fields further down the tree.
		isOverlayField := isOverlayField(structField)
		if isOverlayField {
			retainAncestors = true
		}

		if retain := visitField(ctx, v.Interface()); !isOverlayField && !retain {
			v.Set(reflect.Zero(v.Type()))
			continue
		}

		retainAncestors = true
	}
	return retainAncestors
}

// isOverlayField returns true if a struct field is tagged as overlay-only.
func isOverlayField(structField reflect.StructField) bool {
	return structField.Tag.Get("source") == "overlay-only"
}

// isZero reports whether v is the zero value for its type. It panics if the
// argument is invalid. The implementation has been copied from the upstream Go
// repo as it has not made its way to a stable Go release yet.
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return math.Float64bits(v.Float()) == 0
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isZero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		// This should never happens, but will act as a safeguard for
		// later, as a default value doesn't makes sense here.
		panic(fmt.Sprintf("unexpected value of type %s passed to isZero", v.Kind().String()))
	}
}
