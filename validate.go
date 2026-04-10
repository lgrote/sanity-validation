// Package validate provides Sanity document validation matching the rules
// enforced by Sanity Studio's @sanity/validation package.
//
// It validates both implicit structural invariants (Layer 1: _key, _type,
// Portable Text structure, type enforcement) and explicit Rule-based checks
// (Layer 2: min, max, regex, custom functions, etc.).
package validate

import "fmt"

// Validate checks a document against its schema and type registry.
// Returns nil if valid, or a slice of structured errors.
func Validate(doc *Document, schema *Schema, types TypeResolver) []Error {
	if doc == nil {
		return []Error{{Path: "", Message: "document is nil", Type: ErrNilDocument, Level: LevelError}}
	}
	if schema == nil {
		return []Error{{Path: "", Message: "schema is nil", Type: ErrNilSchema, Level: LevelError}}
	}

	var errs []Error

	// Document-level required fields.
	if doc.Type == "" {
		errs = append(errs, Error{
			Path: "_type", Message: "document type is empty", Type: ErrMissingRequired,
			Got: "empty", Want: "document type name", Level: LevelError,
		})
	}

	// Sync Document struct fields to Fields map so they go through the
	// normal validation pipeline. This maintains backwards compat with
	// callers who construct Documents directly and set struct fields.
	if doc.Fields == nil {
		doc.Fields = make(map[string]any)
	}
	for _, pair := range [...]struct{ key, val string }{
		{"title", doc.Title},
		{"language", doc.Language},
		{"description", doc.Description},
	} {
		if pair.val != "" {
			if _, exists := doc.Fields[pair.key]; !exists {
				doc.Fields[pair.key] = pair.val
			}
		}
	}

	// Validate document-level fields against schema.
	for _, f := range schema.Fields {
		val := doc.Fields[f.Name]
		validateField(val, f, "fields."+f.Name, types, doc.Fields, doc, &errs)
	}

	// Validate sections.
	for i, sec := range doc.Sections {
		secPath := fmt.Sprintf("sections[%d]", i)

		if sec.Key == "" {
			errs = append(errs, Error{
				Path: secPath, Message: "section missing _key",
				Type: ErrMissingKey, Got: "section without _key", Want: "_key string", Level: LevelError,
			})
		}

		if sec.Type == "" {
			errs = append(errs, Error{
				Path: secPath, Message: "section missing _type",
				Type: ErrMissingType, Got: "section without _type", Want: "_type string", Level: LevelError,
			})
			continue
		}

		// Resolve section schema via TypeResolver.
		if types != nil {
			secSchema := types(sec.Type)
			if secSchema != nil {
				for _, f := range secSchema.Fields {
					val := sec.Fields[f.Name]
					validateField(val, f, fmt.Sprintf("%s.fields.%s", secPath, f.Name), types, sec.Fields, doc, &errs)
				}
			}
		}
	}

	return errs
}
