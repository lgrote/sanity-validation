package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateOneField validates a single field value against its field definition,
// wrapping it in a minimal document and schema.
func validateOneField(val any, f Field) []Error {
	doc := &Document{Type: "test", Fields: map[string]any{f.Name: val}}
	schema := &Schema{Name: "test", Fields: []Field{f}}
	return Validate(doc, schema, nil)
}

// validateOneFieldWithResolver is like validateOneField but accepts a TypeResolver.
func validateOneFieldWithResolver(val any, f Field, resolver TypeResolver) []Error {
	doc := &Document{Type: "test", Fields: map[string]any{f.Name: val}}
	schema := &Schema{Name: "test", Fields: []Field{f}}
	return Validate(doc, schema, resolver)
}

// --- String ---

func TestField_String_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("hello", Field{Name: "name", Type: TypeString})
	assert.Empty(t, errs)
}

func TestField_String_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(42, Field{Name: "name", Type: TypeString})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}

func TestField_String_EmptyWhenRequired(t *testing.T) {
	t.Parallel()
	errs := validateOneField("", Field{Name: "name", Type: TypeString, Required: true})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrMissingRequired, errs[0].Type)
}

func TestField_String_Enum_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("a", Field{Name: "color", Type: TypeString, Options: []string{"a", "b"}})
	assert.Empty(t, errs)
}

func TestField_String_Enum_Invalid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("c", Field{Name: "color", Type: TypeString, Options: []string{"a", "b"}})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrInvalidOption, errs[0].Type)
}

// --- Number ---

func TestField_Number_ValidFloat(t *testing.T) {
	t.Parallel()
	errs := validateOneField(3.14, Field{Name: "price", Type: TypeNumber})
	assert.Empty(t, errs)
}

func TestField_Number_ValidInt(t *testing.T) {
	t.Parallel()
	errs := validateOneField(float64(42), Field{Name: "count", Type: TypeNumber})
	assert.Empty(t, errs)
}

func TestField_Number_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField("42", Field{Name: "count", Type: TypeNumber})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}

// --- Boolean ---

func TestField_Boolean_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField(true, Field{Name: "active", Type: TypeBoolean})
	assert.Empty(t, errs)
}

func TestField_Boolean_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField("true", Field{Name: "active", Type: TypeBoolean})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}

// --- Date ---

func TestField_Date_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("2024-01-15", Field{Name: "published", Type: TypeDate})
	assert.Empty(t, errs)
}

func TestField_Date_Invalid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("01-15-2024", Field{Name: "published", Type: TypeDate})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrInvalidFormat, errs[0].Type)
}

func TestField_Date_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(42, Field{Name: "published", Type: TypeDate})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}

// --- URL ---

func TestField_URL_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("https://example.com", Field{Name: "link", Type: TypeURL})
	assert.Empty(t, errs)
}

func TestField_URL_Invalid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("not a url", Field{Name: "link", Type: TypeURL})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrInvalidFormat, errs[0].Type)
}

func TestField_URL_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField(42, Field{Name: "link", Type: TypeURL})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}

// --- Slug ---

func TestField_Slug_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField(map[string]any{"current": "my-slug"}, Field{Name: "slug", Type: TypeSlug})
	assert.Empty(t, errs)
}

func TestField_Slug_MissingCurrent(t *testing.T) {
	t.Parallel()
	errs := validateOneField(map[string]any{}, Field{Name: "slug", Type: TypeSlug})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrMissingRequired, errs[0].Type)
}

func TestField_Slug_WrongType(t *testing.T) {
	t.Parallel()
	errs := validateOneField("my-slug", Field{Name: "slug", Type: TypeSlug})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}

// --- Geopoint ---

func TestField_Geopoint_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField(map[string]any{"lat": 40.7, "lng": -74.0}, Field{Name: "location", Type: TypeGeopoint})
	assert.Empty(t, errs)
}

func TestField_Geopoint_LatOutOfRange(t *testing.T) {
	t.Parallel()
	errs := validateOneField(map[string]any{"lat": 100.0, "lng": -74.0}, Field{Name: "location", Type: TypeGeopoint})
	require.NotEmpty(t, errs)

	found := false
	for _, e := range errs {
		if e.Type == ErrOutOfRange && e.Path == "fields.location.lat" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected out_of_range error for lat, got: %v", errs)
}

func TestField_Geopoint_LngOutOfRange(t *testing.T) {
	t.Parallel()
	errs := validateOneField(map[string]any{"lat": 40.7, "lng": 200.0}, Field{Name: "location", Type: TypeGeopoint})
	require.NotEmpty(t, errs)

	found := false
	for _, e := range errs {
		if e.Type == ErrOutOfRange && e.Path == "fields.location.lng" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected out_of_range error for lng, got: %v", errs)
}

func TestField_Geopoint_MissingLat(t *testing.T) {
	t.Parallel()
	errs := validateOneField(map[string]any{"lng": -74.0}, Field{Name: "location", Type: TypeGeopoint})
	require.NotEmpty(t, errs)

	found := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired && e.Path == "fields.location.lat" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_required error for lat, got: %v", errs)
}

// --- Required / Optional ---

func TestField_Required_Nil(t *testing.T) {
	t.Parallel()
	errs := validateOneField(nil, Field{Name: "name", Type: TypeString, Required: true})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrMissingRequired, errs[0].Type)
}

func TestField_Required_EmptyString(t *testing.T) {
	t.Parallel()
	errs := validateOneField("", Field{Name: "name", Type: TypeString, Required: true})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrMissingRequired, errs[0].Type)
}

func TestField_Required_EmptyArray(t *testing.T) {
	t.Parallel()
	errs := validateOneField([]any{}, Field{Name: "items", Type: TypeArray, Required: true})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrMissingRequired, errs[0].Type)
}

func TestField_Required_EmptyMap(t *testing.T) {
	t.Parallel()
	errs := validateOneField(map[string]any{}, Field{Name: "meta", Type: TypeObject, Required: true})
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrMissingRequired, errs[0].Type)
}

func TestField_Optional_Nil(t *testing.T) {
	t.Parallel()
	errs := validateOneField(nil, Field{Name: "name", Type: TypeString, Required: false})
	assert.Empty(t, errs)
}

// --- Object ---

func TestField_Object_Valid(t *testing.T) {
	t.Parallel()
	f := Field{
		Name: "meta",
		Type: TypeObject,
		Fields: []Field{
			{Name: "author", Type: TypeString, Required: true},
		},
	}
	errs := validateOneField(map[string]any{"author": "Alice"}, f)
	assert.Empty(t, errs)
}

func TestField_Object_MissingRequired(t *testing.T) {
	t.Parallel()
	f := Field{
		Name: "meta",
		Type: TypeObject,
		Fields: []Field{
			{Name: "author", Type: TypeString, Required: true},
		},
	}
	errs := validateOneField(map[string]any{}, f)
	require.NotEmpty(t, errs)

	found := false
	for _, e := range errs {
		if e.Type == ErrMissingRequired && e.Path == "fields.meta.author" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_required error for nested field 'author', got: %v", errs)
}

func TestField_Object_WrongType(t *testing.T) {
	t.Parallel()
	f := Field{
		Name: "meta",
		Type: TypeObject,
		Fields: []Field{
			{Name: "author", Type: TypeString},
		},
	}
	errs := validateOneField("not an object", f)
	require.NotEmpty(t, errs)
	assert.Equal(t, ErrWrongType, errs[0].Type)
}

// --- Custom type (via TypeResolver) ---

func TestField_CustomType_Resolved(t *testing.T) {
	t.Parallel()
	resolver := func(name string) *Schema {
		if name == "address" {
			return &Schema{
				Name: "address",
				Fields: []Field{
					{Name: "street", Type: TypeString, Required: true},
					{Name: "city", Type: TypeString, Required: true},
				},
			}
		}
		return nil
	}

	f := Field{Name: "address", Type: "address"}
	val := map[string]any{"street": "123 Main St", "city": "Springfield"}
	errs := validateOneFieldWithResolver(val, f, resolver)
	assert.Empty(t, errs, "expected no errors for valid custom type, got: %v", errs)

	// Missing a required field in the custom type.
	val2 := map[string]any{"street": "123 Main St"}
	errs2 := validateOneFieldWithResolver(val2, f, resolver)
	require.NotEmpty(t, errs2)

	found := false
	for _, e := range errs2 {
		if e.Type == ErrMissingRequired && e.Path == "fields.address.city" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected missing_required for nested custom type field 'city', got: %v", errs2)
}

func TestField_CustomType_NotFound(t *testing.T) {
	t.Parallel()
	resolver := func(name string) *Schema {
		return nil // always returns nil
	}

	f := Field{Name: "widget", Type: "unknownWidget"}
	val := map[string]any{"foo": "bar"}
	errs := validateOneFieldWithResolver(val, f, resolver)
	assert.Empty(t, errs, "expected no errors when TypeResolver returns nil (can't validate unknown type)")
}
