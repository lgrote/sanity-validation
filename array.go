package validate

import (
	"fmt"
	"reflect"
)

// validateArray checks an array value against its schema definition.
func validateArray(val any, f Field, path string, types TypeResolver, _ map[string]any, doc *Document, errs *[]Error) {
	arr, ok := val.([]any)
	if !ok {
		// Portable Text arrays (ArrayOf contains "block") may still be plain strings
		// before convertBlockContent runs at upload time. Accept strings for those.
		if _, isStr := val.(string); isStr && isPortableTextOf(f) {
			return
		}
		*errs = append(*errs, Error{
			Path: path, Message: "expected array", Type: ErrWrongType,
			Got: fmt.Sprintf("%T", val), Want: "array", Level: LevelError,
		})
		return
	}

	// Bounds checks. A value of 0 means unconstrained.
	if f.MinItems != nil && *f.MinItems > 0 && len(arr) < *f.MinItems {
		*errs = append(*errs, Error{
			Path: path, Message: fmt.Sprintf("array has %d items, minimum is %d", len(arr), *f.MinItems),
			Type: ErrMinItems, Got: fmt.Sprintf("%d items", len(arr)), Want: fmt.Sprintf(">= %d items", *f.MinItems), Level: LevelError,
		})
	}
	if f.MaxItems != nil && *f.MaxItems > 0 && len(arr) > *f.MaxItems {
		*errs = append(*errs, Error{
			Path: path, Message: fmt.Sprintf("array has %d items, maximum is %d", len(arr), *f.MaxItems),
			Type: ErrMaxItems, Got: fmt.Sprintf("%d items", len(arr)), Want: fmt.Sprintf("<= %d items", *f.MaxItems), Level: LevelError,
		})
	}

	// Determine if items should be objects or primitives.
	expectsObjects := arrayExpectsObjects(f)
	expectsPrimitives := arrayExpectsPrimitives(f)

	keys := make(map[string]bool, len(arr))

	for i, item := range arr {
		itemPath := fmt.Sprintf("%s[%d]", path, i)

		obj, isObj := item.(map[string]any)

		// Object item checks.
		if isObj {
			if expectsPrimitives {
				*errs = append(*errs, Error{
					Path: itemPath, Message: "expected primitive value, got object",
					Type: ErrWrongItemType, Got: "object", Want: primitiveTypeName(f), Level: LevelError,
				})
				continue
			}

			// _key check.
			keyVal, hasKey := obj["_key"]
			if !hasKey {
				*errs = append(*errs, Error{
					Path: itemPath, Message: "missing _key (Sanity requires _key on every object in an array)",
					Type: ErrMissingKey, Got: "object without _key", Want: "object with _key string", Level: LevelError,
				})
			} else if k, ok := keyVal.(string); ok {
				if keys[k] {
					*errs = append(*errs, Error{
						Path: itemPath, Message: fmt.Sprintf("duplicate _key %q", k),
						Type: ErrDuplicateKey, Got: k, Want: "unique _key within array", Level: LevelError,
					})
				}
				keys[k] = true
			}

			// _type check for typed arrays.
			if len(f.Of) > 0 && !expectsPrimitives {
				if _, hasType := obj["_type"]; !hasType {
					*errs = append(*errs, Error{
						Path: itemPath, Message: "missing _type on array item",
						Type: ErrMissingType, Got: "object without _type", Want: "object with _type string", Level: LevelError,
					})
				}
			}

			// Validate item fields against ArrayOf schema.
			validateArrayItemFields(obj, f, itemPath, types, doc, errs)
		} else if expectsObjects {
			// Primitive item where we expected an object.
			*errs = append(*errs, Error{
				Path: itemPath, Message: fmt.Sprintf("expected object, got %T", item),
				Type: ErrWrongItemType, Got: describeValue(item), Want: "object with _type and _key", Level: LevelError,
			})
		}
	}
}

// validateArrayItemFields validates an object array item against the ArrayOf field definitions.
func validateArrayItemFields(obj map[string]any, f Field, path string, types TypeResolver, doc *Document, errs *[]Error) {
	if len(f.Of) == 0 {
		return
	}

	// Single ArrayOf with inline fields — validate directly.
	if len(f.Of) == 1 && len(f.Of[0].Fields) > 0 {
		for _, itemField := range f.Of[0].Fields {
			validateField(obj[itemField.Name], itemField, path+"."+itemField.Name, types, obj, doc, errs)
		}
		return
	}

	// Single ArrayOf with named type — resolve and validate.
	if len(f.Of) == 1 && f.Of[0].Type != "string" && f.Of[0].Type != "number" && types != nil {
		schema := types(f.Of[0].Type)
		if schema != nil {
			for _, sf := range schema.Fields {
				validateField(obj[sf.Name], sf, path+"."+sf.Name, types, obj, doc, errs)
			}
		}
		return
	}

	// Polymorphic: multiple ArrayOf — resolve by _type.
	if len(f.Of) > 1 && types != nil {
		typeName, _ := obj["_type"].(string)
		if typeName != "" {
			schema := types(typeName)
			if schema != nil {
				for _, sf := range schema.Fields {
					validateField(obj[sf.Name], sf, path+"."+sf.Name, types, obj, doc, errs)
				}
			}
		}
	}
}

// arrayExpectsObjects returns true if the array schema defines object item types.
func arrayExpectsObjects(f Field) bool {
	if len(f.Of) == 0 {
		return false
	}
	for _, item := range f.Of {
		if item.Type != "string" && item.Type != "number" && item.Type != "boolean" {
			return true
		}
	}
	return false
}

// arrayExpectsPrimitives returns true if the array schema defines only primitive item types.
func arrayExpectsPrimitives(f Field) bool {
	if len(f.Of) == 0 {
		return false
	}
	for _, item := range f.Of {
		if item.Type != "string" && item.Type != "number" && item.Type != "boolean" {
			return false
		}
	}
	return true
}

// isPortableTextOf returns true if the array field's Of contains a "block" type.
func isPortableTextOf(f Field) bool {
	for _, item := range f.Of {
		if item.Type == "block" {
			return true
		}
	}
	return false
}

func primitiveTypeName(f Field) string {
	if len(f.Of) == 1 {
		return f.Of[0].Type
	}
	return "primitive"
}

// deepEqualExcludeKey compares two values for deep equality, excluding _key fields.
// Used by the Unique rule.
func deepEqualExcludeKey(a, b any) bool {
	am, aIsMap := a.(map[string]any)
	bm, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		// Compare maps excluding _key.
		if len(am)-countKey(am) != len(bm)-countKey(bm) {
			return false
		}
		for k, av := range am {
			if k == "_key" {
				continue
			}
			bv, ok := bm[k]
			if !ok || !reflect.DeepEqual(av, bv) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}

func countKey(m map[string]any) int {
	if _, ok := m["_key"]; ok {
		return 1
	}
	return 0
}
