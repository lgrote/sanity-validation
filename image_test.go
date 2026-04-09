package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImage_ValidPlaceholder(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{"_type": "image", "alt": "desc"},
		Field{Name: "hero", Type: TypeImage},
	)
	assert.Empty(t, errs)
}

func TestImage_ValidWithAsset(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{
			"_type": "image",
			"asset": map[string]any{"_type": "reference", "_ref": "image-abc"},
		},
		Field{Name: "hero", Type: TypeImage},
	)
	assert.Empty(t, errs)
}

func TestImage_MissingType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{},
		Field{Name: "hero", Type: TypeImage},
	)
	assert.NotEmpty(t, errs)
	hasMissing := false
	for _, e := range errs {
		if e.Type == ErrMissingType {
			hasMissing = true
		}
	}
	assert.True(t, hasMissing, "expected missing_type error")
}

func TestImage_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		"not an image",
		Field{Name: "hero", Type: TypeImage},
	)
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}

func TestImage_InvalidAsset_MissingRefType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{
			"_type": "image",
			"asset": map[string]any{"_ref": "abc"},
		},
		Field{Name: "hero", Type: TypeImage},
	)
	assert.NotEmpty(t, errs)
	hasMissing := false
	for _, e := range errs {
		if e.Type == ErrMissingType {
			hasMissing = true
		}
	}
	assert.True(t, hasMissing, "expected missing_type error for asset")
}

func TestImage_InvalidAsset_MissingRef(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{
			"_type": "image",
			"asset": map[string]any{"_type": "reference"},
		},
		Field{Name: "hero", Type: TypeImage},
	)
	assert.NotEmpty(t, errs)
	hasRequired := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired {
			hasRequired = true
		}
	}
	assert.True(t, hasRequired, "expected missing_required error for asset _ref")
}

func TestReference_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{"_type": "reference", "_ref": "doc-123"},
		Field{Name: "ref", Type: TypeReference},
	)
	assert.Empty(t, errs)
}

func TestReference_MissingType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{"_ref": "doc-123"},
		Field{Name: "ref", Type: TypeReference},
	)
	assert.NotEmpty(t, errs)
	hasMissing := false
	for _, e := range errs {
		if e.Type == ErrMissingType {
			hasMissing = true
		}
	}
	assert.True(t, hasMissing, "expected missing_type error")
}

func TestReference_MissingRef(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{"_type": "reference"},
		Field{Name: "ref", Type: TypeReference},
	)
	assert.NotEmpty(t, errs)
	hasRequired := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired {
			hasRequired = true
		}
	}
	assert.True(t, hasRequired, "expected missing_required error for _ref")
}

func TestReference_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		"not a ref",
		Field{Name: "ref", Type: TypeReference},
	)
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}
