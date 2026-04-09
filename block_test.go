package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validBlock() map[string]any {
	return map[string]any{
		"_type": "block",
		"_key":  "b1",
		"style": "normal",
		"children": []any{
			map[string]any{"_type": "span", "_key": "s1", "text": "Hello"},
		},
		"markDefs": []any{},
	}
}

func TestBlock_ValidBlock(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{validBlock()},
		Field{Name: "body", Type: TypeBlock},
	)
	assert.Empty(t, errs)
}

func TestBlock_PlainString(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		"some plain text",
		Field{Name: "body", Type: TypeBlock},
	)
	assert.Empty(t, errs)
}

func TestBlock_MissingType(t *testing.T) {
	t.Parallel()
	block := validBlock()
	delete(block, "_type")
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasMissing := false
	for _, e := range errs {
		if e.Type == ErrMissingType {
			hasMissing = true
		}
	}
	assert.True(t, hasMissing, "expected missing_type error")
}

func TestBlock_MissingKey(t *testing.T) {
	t.Parallel()
	block := validBlock()
	delete(block, "_key")
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasMissing := false
	for _, e := range errs {
		if e.Type == ErrMissingKey {
			hasMissing = true
		}
	}
	assert.True(t, hasMissing, "expected missing_key error")
}

func TestBlock_MissingChildren(t *testing.T) {
	t.Parallel()
	block := validBlock()
	delete(block, "children")
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasRequired := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired {
			hasRequired = true
		}
	}
	assert.True(t, hasRequired, "expected missing_required error")
}

func TestBlock_MissingStyle(t *testing.T) {
	t.Parallel()
	block := validBlock()
	delete(block, "style")
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasRequired := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired {
			hasRequired = true
		}
	}
	assert.True(t, hasRequired, "expected missing_required error for style")
}

func TestBlock_EmptyChildren(t *testing.T) {
	t.Parallel()
	block := validBlock()
	block["children"] = []any{}
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasRequired := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired {
			hasRequired = true
		}
	}
	assert.True(t, hasRequired, "expected missing_required error for empty children")
}

func TestBlock_NonArrayChildren(t *testing.T) {
	t.Parallel()
	block := validBlock()
	block["children"] = "text"
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasWrong := false
	for _, e := range errs {
		if e.Type == ErrWrongType {
			hasWrong = true
		}
	}
	assert.True(t, hasWrong, "expected wrong_type error for non-array children")
}

func TestBlock_Span_MissingType(t *testing.T) {
	t.Parallel()
	block := validBlock()
	block["children"] = []any{
		map[string]any{"_key": "s1", "text": "Hello"},
	}
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasMissing := false
	for _, e := range errs {
		if e.Type == ErrMissingType {
			hasMissing = true
		}
	}
	assert.True(t, hasMissing, "expected missing_type error for span")
}

func TestBlock_Span_MissingKey(t *testing.T) {
	t.Parallel()
	block := validBlock()
	block["children"] = []any{
		map[string]any{"_type": "span", "text": "Hello"},
	}
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasMissing := false
	for _, e := range errs {
		if e.Type == ErrMissingKey {
			hasMissing = true
		}
	}
	assert.True(t, hasMissing, "expected missing_key error for span")
}

func TestBlock_Span_MissingText(t *testing.T) {
	t.Parallel()
	block := validBlock()
	block["children"] = []any{
		map[string]any{"_type": "span", "_key": "s1"},
	}
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasRequired := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired {
			hasRequired = true
		}
	}
	assert.True(t, hasRequired, "expected missing_required error for span text")
}

func TestBlock_MarkDef_Valid(t *testing.T) {
	t.Parallel()
	block := validBlock()
	block["markDefs"] = []any{
		map[string]any{"_type": "link", "_key": "md1", "href": "https://example.com"},
	}
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.Empty(t, errs)
}

func TestBlock_MarkDef_MissingType(t *testing.T) {
	t.Parallel()
	block := validBlock()
	block["markDefs"] = []any{
		map[string]any{"_key": "md1"},
	}
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasMissing := false
	for _, e := range errs {
		if e.Type == ErrMissingType {
			hasMissing = true
		}
	}
	assert.True(t, hasMissing, "expected missing_type error for markDef")
}

func TestBlock_MarkDef_MissingKey(t *testing.T) {
	t.Parallel()
	block := validBlock()
	block["markDefs"] = []any{
		map[string]any{"_type": "link"},
	}
	errs := validateOneField([]any{block}, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	hasMissing := false
	for _, e := range errs {
		if e.Type == ErrMissingKey {
			hasMissing = true
		}
	}
	assert.True(t, hasMissing, "expected missing_key error for markDef")
}

func TestBlock_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(42, Field{Name: "body", Type: TypeBlock})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}
