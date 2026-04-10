package validate

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Validator holds parsed schemas from a sanity schema extract and validates
// JSON documents against them.
type Validator struct {
	schemas map[string]*Schema
}

// NewValidator parses the JSON output of `sanity schema extract` and returns
// a Validator that can validate raw Sanity documents.
func NewValidator(data []byte) (*Validator, error) {
	var entries []sanitySchemaEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}

	v := &Validator{schemas: make(map[string]*Schema, len(entries))}

	for _, entry := range entries {
		var attrs map[string]*sanityAttribute

		switch entry.Type {
		case "document":
			attrs = entry.Attributes
		case "type":
			if entry.Value != nil {
				attrs = entry.Value.Attributes
			}
		default:
			continue
		}

		if attrs == nil {
			continue
		}

		fields := convertAttributes(attrs)
		slices.SortFunc(fields, func(a, b Field) int {
			return strings.Compare(a.Name, b.Name)
		})

		v.schemas[entry.Name] = &Schema{
			Name:   entry.Name,
			Fields: fields,
		}
	}

	return v, nil
}

// Schema returns the schema for a given type name, or nil if not found.
func (v *Validator) Schema(name string) *Schema {
	return v.schemas[name]
}

// Resolver returns a TypeResolver backed by all parsed schemas.
func (v *Validator) Resolver() TypeResolver {
	return func(name string) *Schema {
		return v.schemas[name]
	}
}

// Require marks fields as required on a schema. This is needed because
// `sanity schema extract` does not include validation rules.
func (v *Validator) Require(typeName string, fields ...string) {
	schema := v.schemas[typeName]
	if schema == nil {
		return
	}
	for i := range schema.Fields {
		if slices.Contains(fields, schema.Fields[i].Name) {
			schema.Fields[i].Required = true
		}
	}
}

// AddRule adds a validation rule to a field. This is needed because
// `sanity schema extract` does not include validation rules.
func (v *Validator) AddRule(typeName, fieldName string, r Rule) {
	schema := v.schemas[typeName]
	if schema == nil {
		return
	}
	if r.Regex != "" && r.CompiledRegex == nil {
		r.CompiledRegex, _ = regexp.Compile(r.Regex)
	}
	for i := range schema.Fields {
		if schema.Fields[i].Name == fieldName {
			schema.Fields[i].Rules = append(schema.Fields[i].Rules, r)
			return
		}
	}
}

// ParseDocument parses a raw Sanity API JSON document into a Document.
func ParseDocument(data []byte) (*Document, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse document: %w", err)
	}

	doc := &Document{
		Fields: make(map[string]any, len(raw)),
	}
	doc.ID, _ = raw["_id"].(string)
	doc.Type, _ = raw["_type"].(string)
	doc.Title, _ = raw["title"].(string)
	doc.Language, _ = raw["language"].(string)
	doc.Description, _ = raw["description"].(string)

	for k, v := range raw {
		if strings.HasPrefix(k, "_") {
			continue
		}
		// Skip fields already extracted into Document struct fields.
		switch k {
		case "title", "language", "description":
			continue
		}
		doc.Fields[k] = v
	}

	return doc, nil
}

// ValidateDocument parses a raw Sanity JSON document and validates it
// against the schema matching its _type field.
//
// Returns *ValidationError if the document has validation issues,
// or a plain error for parse/schema failures. Returns nil if valid.
func (v *Validator) ValidateDocument(data []byte) error {
	doc, err := ParseDocument(data)
	if err != nil {
		return err
	}

	schema := v.schemas[doc.Type]
	if schema == nil {
		return fmt.Errorf("unknown document type %q", doc.Type)
	}

	errs := Validate(doc, schema, v.Resolver())
	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

// --- Sanity schema extract JSON types ---

type sanitySchemaEntry struct {
	Name       string                      `json:"name"`
	Type       string                      `json:"type"`       // "document" or "type"
	Attributes map[string]*sanityAttribute `json:"attributes"` // document types
	Value      *sanityTypeDescriptor       `json:"value"`      // named types (type: "type")
}

type sanityAttribute struct {
	Type     string                `json:"type"` // "objectAttribute"
	Value    *sanityTypeDescriptor `json:"value"`
	Optional bool                  `json:"optional"`
}

type sanityTypeDescriptor struct {
	Type       string                      `json:"type"` // string, number, boolean, array, object, inline, union, unknown
	Value      any                         `json:"value"`
	Name       string                      `json:"name"` // inline type name
	Of         json.RawMessage             `json:"of"`
	Attributes map[string]*sanityAttribute `json:"attributes"`
	Rest       *sanityTypeDescriptor       `json:"rest"`
}

// --- Internal conversion logic ---

var systemFields = map[string]bool{
	"_id": true, "_type": true, "_rev": true,
	"_createdAt": true, "_updatedAt": true, "_key": true,
}

func convertAttributes(attrs map[string]*sanityAttribute) []Field {
	var fields []Field
	for name, attr := range attrs {
		if systemFields[name] || attr == nil || attr.Value == nil {
			continue
		}
		f := convertTypeDescriptor(name, attr.Value)
		if f != nil {
			fields = append(fields, *f)
		}
	}
	return fields
}

func convertTypeDescriptor(name string, td *sanityTypeDescriptor) *Field {
	if td == nil {
		return nil
	}

	f := &Field{Name: name}

	switch td.Type {
	case "string":
		f.Type = TypeString
	case "number":
		f.Type = TypeNumber
	case "boolean":
		f.Type = TypeBoolean
	case "array":
		f.Type = TypeArray
		convertArrayField(f, td)
	case "object":
		convertObjectField(f, td)
	case "inline":
		f.Type = FieldType(td.Name)
	case "union":
		convertUnionField(f, td)
	default:
		return nil
	}

	return f
}

func convertObjectField(f *Field, td *sanityTypeDescriptor) {
	typeName := extractTypeLiteral(td)
	switch typeName {
	case "image":
		f.Type = TypeImage
		return
	case "reference":
		f.Type = TypeReference
		return
	}

	if td.Rest != nil && td.Rest.Type == "inline" {
		f.Type = FieldType(td.Rest.Name)
		return
	}

	f.Type = TypeObject
	if td.Attributes != nil {
		f.Fields = convertAttributes(td.Attributes)
		slices.SortFunc(f.Fields, func(a, b Field) int {
			return strings.Compare(a.Name, b.Name)
		})
	}
}

func convertArrayField(f *Field, td *sanityTypeDescriptor) {
	if len(td.Of) == 0 {
		return
	}

	var ofDesc sanityTypeDescriptor
	if err := json.Unmarshal(td.Of, &ofDesc); err != nil {
		return
	}

	if ofDesc.Type == "union" {
		convertArrayUnion(f, &ofDesc)
	} else {
		item := convertArrayItem(&ofDesc)
		if item != nil {
			if item.Type == "block" {
				f.Type = TypeBlock
				return
			}
			f.Of = []ArrayItem{*item}
		}
	}
}

func convertArrayUnion(f *Field, unionDesc *sanityTypeDescriptor) {
	if len(unionDesc.Of) == 0 {
		return
	}

	var members []sanityTypeDescriptor
	if err := json.Unmarshal(unionDesc.Of, &members); err != nil {
		return
	}

	hasBlock := false
	var items []ArrayItem

	for i := range members {
		item := convertArrayItem(&members[i])
		if item == nil {
			continue
		}
		if item.Type == "block" {
			hasBlock = true
		}
		items = append(items, *item)
	}

	if hasBlock && len(items) == 1 {
		f.Type = TypeBlock
		return
	}

	f.Of = items
}

func convertArrayItem(td *sanityTypeDescriptor) *ArrayItem {
	switch td.Type {
	case "string":
		return &ArrayItem{Type: "string"}
	case "number":
		return &ArrayItem{Type: "number"}
	case "boolean":
		return &ArrayItem{Type: "boolean"}
	case "inline":
		return &ArrayItem{Type: td.Name}
	case "object":
		if td.Rest != nil && td.Rest.Type == "inline" {
			return &ArrayItem{Type: td.Rest.Name}
		}
		if _, hasChildren := td.Attributes["children"]; hasChildren {
			return &ArrayItem{Type: "block"}
		}
		typeName := extractTypeLiteral(td)
		if typeName != "" {
			return &ArrayItem{Type: typeName}
		}
		fields := convertAttributes(td.Attributes)
		slices.SortFunc(fields, func(a, b Field) int {
			return strings.Compare(a.Name, b.Name)
		})
		return &ArrayItem{Type: "object", Fields: fields}
	}
	return nil
}

func convertUnionField(f *Field, td *sanityTypeDescriptor) {
	if len(td.Of) == 0 {
		f.Type = TypeString
		return
	}

	var members []sanityTypeDescriptor
	if err := json.Unmarshal(td.Of, &members); err != nil {
		f.Type = TypeString
		return
	}

	var options []string
	for _, m := range members {
		if m.Type != "string" {
			f.Type = TypeString
			return
		}
		if s, ok := m.Value.(string); ok {
			options = append(options, s)
		}
	}

	f.Type = TypeString
	if len(options) > 0 {
		f.Options = options
	}
}

func extractTypeLiteral(td *sanityTypeDescriptor) string {
	if td == nil || td.Attributes == nil {
		return ""
	}
	typeAttr, ok := td.Attributes["_type"]
	if !ok || typeAttr == nil || typeAttr.Value == nil {
		return ""
	}
	if typeAttr.Value.Type == "string" {
		s, _ := typeAttr.Value.Value.(string)
		return s
	}
	return ""
}
