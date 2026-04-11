package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArray_Valid_WithKeys(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{map[string]any{"_key": "k1", "_type": "item", "title": "X"}},
		Field{Name: "items", Type: TypeArray, Of: []ArrayItem{{Type: "item"}}},
	)
	assert.Empty(t, errs)
}

func TestArray_MissingKey(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{map[string]any{"_type": "item", "title": "X"}},
		Field{Name: "items", Type: TypeArray, Of: []ArrayItem{{Type: "item"}}},
	)
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrMissingKey, errs[0].Type)
}

func TestArray_MissingType_TypedArray(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{map[string]any{"_key": "k1", "title": "X"}},
		Field{Name: "items", Type: TypeArray, Of: []ArrayItem{{Type: "item"}}},
	)
	assert.NotEmpty(t, errs)
	hasType := false
	for _, e := range errs {
		if e.Type == ErrMissingType {
			hasType = true
		}
	}
	assert.True(t, hasType, "expected missing_type error")
}

// TestArray_AnonymousInline_DoesNotRequireType covers the regression
// where overviewSection.items (and every similar anonymous-inline
// array) was flagged as missing _type even though Sanity itself
// rejects any _type value on these members. The schema shape is
// Of: [{Type: "object", Fields: [...]}] — a single ArrayOf entry
// with inline fields. Items carry _key only; no _type at all.
func TestArray_AnonymousInline_DoesNotRequireType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{
			map[string]any{"_key": "k1", "title": "X", "description": "Y"},
			map[string]any{"_key": "k2", "title": "A", "description": "B"},
		},
		Field{
			Name: "items", Type: TypeArray,
			Of: []ArrayItem{
				{
					Type: "object",
					Fields: []Field{
						{Name: "title", Type: TypeString},
						{Name: "description", Type: TypeString},
					},
				},
			},
		},
	)
	for _, e := range errs {
		assert.NotEqual(t, ErrMissingType, e.Type,
			"anonymous inline array items must not be flagged as missing _type")
	}
}

// TestArray_NamedType_StillRequiresType guards against over-correcting
// the anonymous-inline fix. Named-type single ArrayOf entries (no inline
// Fields) still need _type pinned to the registered type name.
func TestArray_NamedType_StillRequiresType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{map[string]any{"_key": "k1", "title": "X"}},
		Field{
			Name: "items", Type: TypeArray,
			Of: []ArrayItem{{Type: "quickFact"}},
		},
	)
	hasType := false
	for _, e := range errs {
		if e.Type == ErrMissingType {
			hasType = true
		}
	}
	assert.True(t, hasType,
		"named-type array items must still be flagged when _type is missing")
}

func TestArray_DuplicateKeys(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{
			map[string]any{"_key": "k1", "_type": "item"},
			map[string]any{"_key": "k1", "_type": "item"},
		},
		Field{Name: "items", Type: TypeArray, Of: []ArrayItem{{Type: "item"}}},
	)
	assert.NotEmpty(t, errs)
	hasDup := false
	for _, e := range errs {
		if e.Type == ErrDuplicateKey {
			hasDup = true
		}
	}
	assert.True(t, hasDup, "expected duplicate_key error")
}

func TestArray_PrimitiveArray_ValidStrings(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{"a", "b"},
		Field{Name: "tags", Type: TypeArray, Of: []ArrayItem{{Type: "string"}}},
	)
	assert.Empty(t, errs)
}

func TestArray_PrimitiveArray_RejectsObjects(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{map[string]any{"_key": "k1", "_type": "item"}},
		Field{Name: "tags", Type: TypeArray, Of: []ArrayItem{{Type: "string"}}},
	)
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongItemType, errs[0].Type)
}

func TestArray_ObjectArray_RejectsStrings(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{"string"},
		Field{Name: "items", Type: TypeArray, Of: []ArrayItem{{Type: "item"}}},
	)
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongItemType, errs[0].Type)
}

func TestArray_MinItems(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{map[string]any{"_key": "k1", "_type": "item"}},
		Field{Name: "items", Type: TypeArray, Of: []ArrayItem{{Type: "item"}}, MinItems: new(2)},
	)
	assert.NotEmpty(t, errs)
	hasMin := false
	for _, e := range errs {
		if e.Type == ErrMinItems {
			hasMin = true
		}
	}
	assert.True(t, hasMin, "expected min_items error")
}

func TestArray_MaxItems(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{
			map[string]any{"_key": "k1", "_type": "item"},
			map[string]any{"_key": "k2", "_type": "item"},
			map[string]any{"_key": "k3", "_type": "item"},
		},
		Field{Name: "items", Type: TypeArray, Of: []ArrayItem{{Type: "item"}}, MaxItems: new(2)},
	)
	assert.NotEmpty(t, errs)
	hasMax := false
	for _, e := range errs {
		if e.Type == ErrMaxItems {
			hasMax = true
		}
	}
	assert.True(t, hasMax, "expected max_items error")
}

func TestArray_WithinBounds(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{
			map[string]any{"_key": "k1", "_type": "item"},
			map[string]any{"_key": "k2", "_type": "item"},
		},
		Field{Name: "items", Type: TypeArray, Of: []ArrayItem{{Type: "item"}}, MinItems: new(1), MaxItems: new(3)},
	)
	assert.Empty(t, errs)
}

func TestArray_EmptyWhenRequired(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{},
		Field{Name: "items", Type: TypeArray, Required: true, Of: []ArrayItem{{Type: "item"}}},
	)
	assert.NotEmpty(t, errs)
	hasRequired := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired {
			hasRequired = true
		}
	}
	assert.True(t, hasRequired, "expected missing_required error")
}

func TestArray_InlineFieldValidation(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{map[string]any{"_key": "k1", "_type": "item", "title": ""}},
		Field{
			Name: "items", Type: TypeArray,
			Of: []ArrayItem{{
				Type: "item",
				Fields: []Field{
					{Name: "title", Type: TypeString, Required: true},
				},
			}},
		},
	)
	assert.NotEmpty(t, errs)
	hasRequired := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired {
			hasRequired = true
		}
	}
	assert.True(t, hasRequired, "expected missing_required error for inline field")
}

func TestArray_NamedTypeValidation(t *testing.T) {
	t.Parallel()
	resolver := func(name string) *Schema {
		if name == "feature" {
			return &Schema{
				Name: "feature",
				Fields: []Field{
					{Name: "label", Type: TypeString, Required: true},
				},
			}
		}
		return nil
	}

	doc := &Document{Type: "test", Fields: map[string]any{
		"items": []any{map[string]any{"_key": "k1", "_type": "feature"}},
	}}
	schema := &Schema{Name: "test", Fields: []Field{
		{Name: "items", Type: TypeArray, Of: []ArrayItem{{Type: "feature"}}},
	}}

	errs := Validate(doc, schema, resolver)
	assert.NotEmpty(t, errs)
	hasRequired := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired {
			hasRequired = true
		}
	}
	assert.True(t, hasRequired, "expected missing_required error via TypeResolver")
}

func TestArray_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		"not an array",
		Field{Name: "items", Type: TypeArray},
	)
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}

func TestDeepEqualExcludeKey_Deterministic(t *testing.T) {
	t.Parallel()
	// Two maps with identical content but different _key values must be equal.
	a := map[string]any{"_key": "k1", "title": "hello", "count": 42.0}
	b := map[string]any{"_key": "k2", "title": "hello", "count": 42.0}
	assert.True(t, deepEqualExcludeKey(a, b))

	// Different content must not be equal.
	c := map[string]any{"_key": "k3", "title": "world", "count": 42.0}
	assert.False(t, deepEqualExcludeKey(a, c))
}
