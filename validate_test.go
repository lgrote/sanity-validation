package validate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_NilDocument(t *testing.T) {
	t.Parallel()
	schema := &Schema{Name: "page"}
	errs := Validate(nil, schema, nil)
	require.Len(t, errs, 1)
	assert.Equal(t, ErrNilDocument, errs[0].Type)
	assert.Contains(t, errs[0].Message, "nil")
}

func TestValidate_NilSchema(t *testing.T) {
	t.Parallel()
	doc := &Document{Type: "page"}
	errs := Validate(doc, nil, nil)
	require.Len(t, errs, 1)
	assert.Equal(t, ErrNilSchema, errs[0].Type)
	assert.Contains(t, errs[0].Message, "nil")
}

func TestValidate_ValidDocument(t *testing.T) {
	t.Parallel()
	resolver := func(name string) *Schema {
		if name == "hero" {
			return &Schema{
				Name: "hero",
				Fields: []Field{
					{Name: "heading", Type: TypeString, Required: true},
				},
			}
		}
		return nil
	}

	doc := &Document{
		ID:          "doc-1",
		Type:        "page",
		Language:    "en",
		Title:       "Test Page",
		Description: "A test page",
		Fields: map[string]any{
			"slug": map[string]any{"current": "test-page"},
		},
		Sections: []Section{
			{
				Type: "hero",
				Key:  "hero-1",
				Fields: map[string]any{
					"heading": "Welcome",
				},
			},
		},
	}

	schema := &Schema{
		Name: "page",
		Fields: []Field{
			{Name: "title", Type: TypeString, Required: true},
			{Name: "language", Type: TypeString, Required: true},
			{Name: "description", Type: TypeString, Required: true},
			{Name: "slug", Type: TypeSlug},
		},
	}

	errs := Validate(doc, schema, resolver)
	assert.Empty(t, errs, "expected no errors for a valid document, got: %v", errs)
}

func TestValidate_MissingType(t *testing.T) {
	t.Parallel()
	doc := &Document{Type: "", Fields: map[string]any{}}
	schema := &Schema{Name: "page"}

	errs := Validate(doc, schema, nil)
	require.NotEmpty(t, errs)

	found := false
	for _, e := range errs {
		if e.Path == "_type" && e.Type == ErrMissingRequired {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_required error at _type, got: %v", errs)
}

func TestValidate_RequiredTitle(t *testing.T) {
	t.Parallel()
	doc := &Document{Type: "page", Title: "", Fields: map[string]any{}}
	schema := &Schema{
		Name:   "page",
		Fields: []Field{{Name: "title", Type: TypeString, Required: true}},
	}

	errs := Validate(doc, schema, nil)
	require.NotEmpty(t, errs)

	var titleErr *Error
	for i := range errs {
		if errs[i].Path == "fields.title" {
			titleErr = &errs[i]
			break
		}
	}
	require.NotNil(t, titleErr, "expected error at path 'fields.title'")
	assert.Equal(t, ErrMissingRequired, titleErr.Type)
}

func TestValidate_RequiredLanguage(t *testing.T) {
	t.Parallel()
	doc := &Document{Type: "page", Language: "", Fields: map[string]any{}}
	schema := &Schema{
		Name:   "page",
		Fields: []Field{{Name: "language", Type: TypeString, Required: true}},
	}

	errs := Validate(doc, schema, nil)
	require.NotEmpty(t, errs)

	var langErr *Error
	for i := range errs {
		if errs[i].Path == "fields.language" {
			langErr = &errs[i]
			break
		}
	}
	require.NotNil(t, langErr, "expected error at path 'fields.language'")
	assert.Equal(t, ErrMissingRequired, langErr.Type)
}

func TestValidate_RequiredDescription(t *testing.T) {
	t.Parallel()
	doc := &Document{Type: "page", Description: "", Fields: map[string]any{}}
	schema := &Schema{
		Name:   "page",
		Fields: []Field{{Name: "description", Type: TypeString, Required: true}},
	}

	errs := Validate(doc, schema, nil)
	require.NotEmpty(t, errs)

	var descErr *Error
	for i := range errs {
		if errs[i].Path == "fields.description" {
			descErr = &errs[i]
			break
		}
	}
	require.NotNil(t, descErr, "expected error at path 'fields.description'")
	assert.Equal(t, ErrMissingRequired, descErr.Type)
}

func TestValidate_SectionMissingKey(t *testing.T) {
	t.Parallel()
	doc := &Document{
		Type:   "page",
		Fields: map[string]any{},
		Sections: []Section{
			{Type: "hero", Key: ""},
		},
	}
	schema := &Schema{Name: "page"}

	errs := Validate(doc, schema, nil)
	require.NotEmpty(t, errs)

	found := false
	for _, e := range errs {
		if e.Type == ErrMissingKey && strings.HasPrefix(e.Path, "sections[") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_key error for section, got: %v", errs)
}

func TestValidate_SectionMissingType(t *testing.T) {
	t.Parallel()
	doc := &Document{
		Type:   "page",
		Fields: map[string]any{},
		Sections: []Section{
			{Type: "", Key: "sec-1"},
		},
	}
	schema := &Schema{Name: "page"}

	errs := Validate(doc, schema, nil)
	require.NotEmpty(t, errs)

	found := false
	for _, e := range errs {
		if e.Type == ErrMissingType && strings.HasPrefix(e.Path, "sections[") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_type error for section, got: %v", errs)
}

func TestValidate_SectionFieldsValidated(t *testing.T) {
	t.Parallel()
	resolver := func(name string) *Schema {
		if name == "hero" {
			return &Schema{
				Name: "hero",
				Fields: []Field{
					{Name: "heading", Type: TypeString, Required: true},
				},
			}
		}
		return nil
	}

	doc := &Document{
		Type:   "page",
		Fields: map[string]any{},
		Sections: []Section{
			{
				Type:   "hero",
				Key:    "hero-1",
				Fields: map[string]any{}, // heading missing
			},
		},
	}
	schema := &Schema{Name: "page"}

	errs := Validate(doc, schema, resolver)
	require.NotEmpty(t, errs)

	found := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired && strings.Contains(e.Path, "heading") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_required error for section field 'heading', got: %v", errs)
}

func TestValidate_TypeResolverCalled(t *testing.T) {
	t.Parallel()
	called := false
	resolver := func(name string) *Schema {
		if name == "cta" {
			called = true
			return &Schema{Name: "cta", Fields: []Field{}}
		}
		return nil
	}

	doc := &Document{
		Type:   "page",
		Fields: map[string]any{},
		Sections: []Section{
			{Type: "cta", Key: "cta-1", Fields: map[string]any{}},
		},
	}
	schema := &Schema{Name: "page"}

	Validate(doc, schema, resolver)
	assert.True(t, called, "expected TypeResolver to be called for section type 'cta'")
}

func TestFormatErrors_Empty(t *testing.T) {
	t.Parallel()
	result := FormatErrors(nil)
	assert.Empty(t, result)

	result = FormatErrors([]Error{})
	assert.Empty(t, result)
}

func TestFormatErrors_MultipleErrors(t *testing.T) {
	t.Parallel()
	errs := []Error{
		{Path: "_type", Message: "document type is empty", Type: ErrMissingRequired, Got: "empty", Want: "document type name"},
		{Path: "title", Message: "required field title is empty", Type: ErrMissingRequired, Got: "empty", Want: "non-empty string"},
		{Path: "fields.slug", Message: "expected slug object", Type: ErrWrongType, Got: "string", Want: `object with "current" string field`},
	}

	result := FormatErrors(errs)
	lines := strings.Split(result, "\n")
	assert.Len(t, lines, 3)

	assert.Contains(t, lines[0], "_type")
	assert.Contains(t, lines[0], "document type is empty")
	assert.Contains(t, lines[0], "got=empty")
	assert.Contains(t, lines[0], "want=document type name")

	assert.Contains(t, lines[1], "title")
	assert.Contains(t, lines[1], "required field title is empty")

	assert.Contains(t, lines[2], "fields.slug")
	assert.Contains(t, lines[2], "expected slug object")
	assert.Contains(t, lines[2], "got=string")
}

func TestValidate_TitleNotDoubleValidated(t *testing.T) {
	t.Parallel()
	// A required "title" field should produce exactly one error, not two.
	doc := &Document{Type: "test", Title: "", Fields: map[string]any{}}
	schema := &Schema{Name: "test", Fields: []Field{
		{Name: "title", Type: TypeString, Required: true},
	}}
	errs := Validate(doc, schema, nil)
	titleErrs := 0
	for _, e := range errs {
		if e.Path == "fields.title" {
			titleErrs++
		}
	}
	assert.Equal(t, 1, titleErrs, "title should produce exactly one error")
}

func TestValidate_TitleRulesEvaluated(t *testing.T) {
	t.Parallel()
	// Rules on title fields should now be evaluated (not bypassed).
	minLen := 10
	doc := &Document{Type: "page", Title: "Hi", Fields: map[string]any{}}
	schema := &Schema{
		Name: "page",
		Fields: []Field{{
			Name: "title", Type: TypeString,
			Rules: []Rule{{Min: &minLen}},
		}},
	}
	errs := Validate(doc, schema, nil)
	var ruleErr *Error
	for i := range errs {
		if errs[i].Type == ErrRuleMin {
			ruleErr = &errs[i]
			break
		}
	}
	require.NotNil(t, ruleErr, "expected min rule error on title")
	assert.Equal(t, "fields.title", ruleErr.Path)
}
