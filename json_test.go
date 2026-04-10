package validate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator_DocumentType(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_id":   {"type": "objectAttribute", "value": {"type": "string"}},
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"title": {"type": "objectAttribute", "value": {"type": "string"}, "optional": true},
				"count": {"type": "objectAttribute", "value": {"type": "number"}, "optional": true},
				"active":{"type": "objectAttribute", "value": {"type": "boolean"}, "optional": true}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("article")
	require.NotNil(t, s)
	assert.Equal(t, "article", s.Name)
	assert.Len(t, s.Fields, 3) // title, count, active (system fields skipped)

	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeString, byName["title"].Type)
	assert.Equal(t, TypeNumber, byName["count"].Type)
	assert.Equal(t, TypeBoolean, byName["active"].Type)
}

func TestNewValidator_ObjectType(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "faqItem",
			"type": "type",
			"value": {
				"type": "object",
				"attributes": {
					"_type":    {"type": "objectAttribute", "value": {"type": "string", "value": "faqItem"}},
					"question": {"type": "objectAttribute", "value": {"type": "string"}, "optional": true},
					"answer":   {"type": "objectAttribute", "value": {"type": "string"}, "optional": true}
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("faqItem")
	require.NotNil(t, s)
	assert.Equal(t, "faqItem", s.Name)
	assert.Len(t, s.Fields, 2)
}

func TestNewValidator_Enum(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"status": {
					"type": "objectAttribute",
					"value": {
						"type": "union",
						"of": [
							{"type": "string", "value": "draft"},
							{"type": "string", "value": "published"},
							{"type": "string", "value": "archived"}
						]
					},
					"optional": true
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	status := byName["status"]
	assert.Equal(t, TypeString, status.Type)
	assert.Equal(t, []string{"draft", "published", "archived"}, status.Options)
}

func TestNewValidator_ArrayOfPrimitives(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"tags": {
					"type": "objectAttribute",
					"value": {"type": "array", "of": {"type": "string"}},
					"optional": true
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	tags := byName["tags"]
	assert.Equal(t, TypeArray, tags.Type)
	require.Len(t, tags.Of, 1)
	assert.Equal(t, "string", tags.Of[0].Type)
}

func TestNewValidator_ArrayOfNamedType(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"items": {
					"type": "objectAttribute",
					"value": {
						"type": "array",
						"of": {
							"type": "object",
							"attributes": {
								"_key": {"type": "objectAttribute", "value": {"type": "string"}}
							},
							"rest": {"type": "inline", "name": "faqItem"}
						}
					},
					"optional": true
				}
			}
		},
		{
			"name": "faqItem",
			"type": "type",
			"value": {
				"type": "object",
				"attributes": {
					"_type":    {"type": "objectAttribute", "value": {"type": "string", "value": "faqItem"}},
					"question": {"type": "objectAttribute", "value": {"type": "string"}, "optional": true}
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	items := byName["items"]
	assert.Equal(t, TypeArray, items.Type)
	require.Len(t, items.Of, 1)
	assert.Equal(t, "faqItem", items.Of[0].Type)

	// Resolver can find faqItem
	resolver := v.Resolver()
	assert.NotNil(t, resolver("faqItem"))
}

func TestNewValidator_PolymorphicArray(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "page",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "page"}},
				"sections": {
					"type": "objectAttribute",
					"value": {
						"type": "array",
						"of": {
							"type": "union",
							"of": [
								{
									"type": "object",
									"attributes": {"_key": {"type": "objectAttribute", "value": {"type": "string"}}},
									"rest": {"type": "inline", "name": "faqSection"}
								},
								{
									"type": "object",
									"attributes": {"_key": {"type": "objectAttribute", "value": {"type": "string"}}},
									"rest": {"type": "inline", "name": "ctaSection"}
								}
							]
						}
					},
					"optional": true
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("page")
	byName := fieldMap(s.Fields)
	sections := byName["sections"]
	assert.Equal(t, TypeArray, sections.Type)
	require.Len(t, sections.Of, 2)
	assert.Equal(t, "faqSection", sections.Of[0].Type)
	assert.Equal(t, "ctaSection", sections.Of[1].Type)
}

func TestNewValidator_ImageDetection(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"logo": {
					"type": "objectAttribute",
					"value": {
						"type": "object",
						"attributes": {
							"asset":   {"type": "objectAttribute", "value": {"type": "inline", "name": "sanity.imageAsset.reference"}, "optional": true},
							"hotspot": {"type": "objectAttribute", "value": {"type": "inline", "name": "sanity.imageHotspot"}, "optional": true},
							"_type":   {"type": "objectAttribute", "value": {"type": "string", "value": "image"}}
						}
					},
					"optional": true
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeImage, byName["logo"].Type)
}

func TestNewValidator_InlineType(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"seo": {
					"type": "objectAttribute",
					"value": {"type": "inline", "name": "seoFields"},
					"optional": true
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.Equal(t, FieldType("seoFields"), byName["seo"].Type)
}

func TestNewValidator_NestedObject(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"prosCons": {
					"type": "objectAttribute",
					"value": {
						"type": "object",
						"attributes": {
							"pros": {
								"type": "objectAttribute",
								"value": {"type": "array", "of": {"type": "string"}},
								"optional": true
							},
							"cons": {
								"type": "objectAttribute",
								"value": {"type": "array", "of": {"type": "string"}},
								"optional": true
							}
						}
					},
					"optional": true
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	pc := byName["prosCons"]
	assert.Equal(t, TypeObject, pc.Type)
	assert.Len(t, pc.Fields, 2)

	nested := fieldMap(pc.Fields)
	assert.Equal(t, TypeArray, nested["pros"].Type)
	assert.Equal(t, TypeArray, nested["cons"].Type)
}

func TestNewValidator_PortableText(t *testing.T) {
	t.Parallel()

	// Array with sole block-type member → TypeBlock
	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"body": {
					"type": "objectAttribute",
					"value": {
						"type": "array",
						"of": {
							"type": "object",
							"attributes": {
								"children": {"type": "objectAttribute", "value": {"type": "array", "of": {"type": "object", "attributes": {}}}},
								"style":    {"type": "objectAttribute", "value": {"type": "string"}, "optional": true},
								"markDefs": {"type": "objectAttribute", "value": {"type": "array", "of": {"type": "object", "attributes": {}}}, "optional": true},
								"_type":    {"type": "objectAttribute", "value": {"type": "string", "value": "block"}}
							},
							"rest": {
								"type": "object",
								"attributes": {
									"_key": {"type": "objectAttribute", "value": {"type": "string"}}
								}
							}
						}
					},
					"optional": true
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeBlock, byName["body"].Type)
}

func TestNewValidator_MixedBlockArray(t *testing.T) {
	t.Parallel()

	// Array with block + image → stays TypeArray with Of items
	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"body": {
					"type": "objectAttribute",
					"value": {
						"type": "array",
						"of": {
							"type": "union",
							"of": [
								{
									"type": "object",
									"attributes": {
										"children": {"type": "objectAttribute", "value": {"type": "array", "of": {"type": "object", "attributes": {}}}},
										"_type":    {"type": "objectAttribute", "value": {"type": "string", "value": "block"}}
									},
									"rest": {"type": "object", "attributes": {"_key": {"type": "objectAttribute", "value": {"type": "string"}}}}
								},
								{
									"type": "object",
									"attributes": {
										"asset": {"type": "objectAttribute", "value": {"type": "inline", "name": "sanity.imageAsset.reference"}, "optional": true},
										"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "image"}}
									}
								}
							]
						}
					},
					"optional": true
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	body := byName["body"]
	assert.Equal(t, TypeArray, body.Type)
	require.Len(t, body.Of, 2)
	assert.Equal(t, "block", body.Of[0].Type)
	assert.Equal(t, "image", body.Of[1].Type)
}

func TestNewValidator_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := NewValidator([]byte(`not json`))
	assert.Error(t, err)
}

func TestParseDocument(t *testing.T) {
	t.Parallel()

	docJSON := `{
		"_id": "brand-123",
		"_type": "brand",
		"_rev": "abc",
		"_createdAt": "2024-01-01T00:00:00Z",
		"_updatedAt": "2024-01-01T00:00:00Z",
		"language": "en",
		"title": "Hertz Review",
		"description": "A review of Hertz",
		"name": "Hertz",
		"rating": 4.2,
		"tags": ["rental", "car"]
	}`

	doc, err := ParseDocument([]byte(docJSON))
	require.NoError(t, err)

	assert.Equal(t, "brand-123", doc.ID)
	assert.Equal(t, "brand", doc.Type)
	assert.Equal(t, "en", doc.Language)
	assert.Equal(t, "Hertz Review", doc.Title)
	assert.Equal(t, "A review of Hertz", doc.Description)

	// System fields excluded from Fields
	assert.Nil(t, doc.Fields["_id"])
	assert.Nil(t, doc.Fields["_type"])
	assert.Nil(t, doc.Fields["_rev"])

	// Regular fields present
	assert.Equal(t, "Hertz", doc.Fields["name"])
	assert.InDelta(t, 4.2, doc.Fields["rating"], 0.001)
	assert.Len(t, doc.Fields["tags"], 2)

	// title/language/description are on both the struct and in Fields
	assert.Equal(t, "Hertz Review", doc.Fields["title"])
	assert.Equal(t, "en", doc.Fields["language"])
	assert.Equal(t, "A review of Hertz", doc.Fields["description"])
}

func TestParseDocument_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseDocument([]byte(`not json`))
	assert.Error(t, err)
}

func TestRequire(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"title": {"type": "objectAttribute", "value": {"type": "string"}, "optional": true},
				"body":  {"type": "objectAttribute", "value": {"type": "string"}, "optional": true}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	v.Require("article", "title", "body")

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.True(t, byName["title"].Required)
	assert.True(t, byName["body"].Required)
}

func TestAddRule(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type":  {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"rating": {"type": "objectAttribute", "value": {"type": "number"}, "optional": true}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	ruleMin, ruleMax := 0, 5
	require.NoError(t, v.AddRule("article", "rating", Rule{Min: &ruleMin, Max: &ruleMax}))

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	require.Len(t, byName["rating"].Rules, 1)
	assert.Equal(t, 0, *byName["rating"].Rules[0].Min)
	assert.Equal(t, 5, *byName["rating"].Rules[0].Max)
}

func TestValidateDocument_EndToEnd(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "article",
			"type": "document",
			"attributes": {
				"_type":  {"type": "objectAttribute", "value": {"type": "string", "value": "article"}},
				"name":   {"type": "objectAttribute", "value": {"type": "string"}, "optional": true},
				"rating": {"type": "objectAttribute", "value": {"type": "number"}, "optional": true},
				"status": {
					"type": "objectAttribute",
					"value": {
						"type": "union",
						"of": [
							{"type": "string", "value": "draft"},
							{"type": "string", "value": "published"}
						]
					},
					"optional": true
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	v.Require("article", "name")
	ruleMin := 0
	ruleMax := 5
	require.NoError(t, v.AddRule("article", "rating", Rule{Min: &ruleMin, Max: &ruleMax}))

	// Valid document
	validDoc := `{"_id": "1", "_type": "article", "name": "Test", "rating": 3, "status": "draft"}`
	require.NoError(t, v.ValidateDocument([]byte(validDoc)))

	// Missing required field
	missingName := `{"_id": "2", "_type": "article", "rating": 3}`
	err = v.ValidateDocument([]byte(missingName))
	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, ErrMissingRequired, ve.Errors[0].Type)

	// Invalid enum value
	badEnum := `{"_id": "3", "_type": "article", "name": "Test", "status": "invalid"}`
	err = v.ValidateDocument([]byte(badEnum))
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, ErrInvalidOption, ve.Errors[0].Type)

	// Rating out of range
	badRating := `{"_id": "4", "_type": "article", "name": "Test", "rating": 10}`
	require.Error(t, v.ValidateDocument([]byte(badRating)))

	// Wrong type
	wrongType := `{"_id": "5", "_type": "article", "name": "Test", "rating": "not a number"}`
	err = v.ValidateDocument([]byte(wrongType))
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, ErrWrongType, ve.Errors[0].Type)
}

func TestValidateDocument_UnknownType(t *testing.T) {
	t.Parallel()

	v, err := NewValidator([]byte(`[]`))
	require.NoError(t, err)

	doc := `{"_id": "1", "_type": "nonexistent"}`
	err = v.ValidateDocument([]byte(doc))
	require.Error(t, err)
	// Not a ValidationError — it's a schema lookup failure
	var ve *ValidationError
	assert.NotErrorAs(t, err, &ve)
}

func TestValidateDocument_TypeResolution(t *testing.T) {
	t.Parallel()

	schema := `[
		{
			"name": "page",
			"type": "document",
			"attributes": {
				"_type": {"type": "objectAttribute", "value": {"type": "string", "value": "page"}},
				"items": {
					"type": "objectAttribute",
					"value": {
						"type": "array",
						"of": {
							"type": "object",
							"attributes": {
								"_key": {"type": "objectAttribute", "value": {"type": "string"}}
							},
							"rest": {"type": "inline", "name": "faqItem"}
						}
					},
					"optional": true
				}
			}
		},
		{
			"name": "faqItem",
			"type": "type",
			"value": {
				"type": "object",
				"attributes": {
					"_type":    {"type": "objectAttribute", "value": {"type": "string", "value": "faqItem"}},
					"question": {"type": "objectAttribute", "value": {"type": "string"}, "optional": true}
				}
			}
		}
	]`

	v, err := NewValidator([]byte(schema))
	require.NoError(t, err)

	v.Require("faqItem", "question")

	// Valid: array item has required question field
	validDoc := `{
		"_id": "1", "_type": "page",
		"items": [
			{"_type": "faqItem", "_key": "k1", "question": "How?"}
		]
	}`
	require.NoError(t, v.ValidateDocument([]byte(validDoc)))

	// Invalid: array item missing required question field
	invalidDoc := `{
		"_id": "2", "_type": "page",
		"items": [
			{"_type": "faqItem", "_key": "k1"}
		]
	}`
	assert.Error(t, v.ValidateDocument([]byte(invalidDoc)))
}

func TestNewValidator_RealSchemaExtract(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("/Users/lgr/workspace/go/src/github.com/lgrote/websites/dev.pricecomparecar.com/studio/schema.json")
	if err != nil {
		t.Skip("real schema file not available")
	}

	v, err := NewValidator(data)
	require.NoError(t, err)

	// Document types should be parsed
	brand := v.Schema("brand")
	require.NotNil(t, brand, "brand document type should exist")

	byName := fieldMap(brand.Fields)

	// Basic field types
	assert.Equal(t, TypeString, byName["name"].Type, "name should be string")
	assert.Equal(t, TypeNumber, byName["rating"].Type, "rating should be number")
	assert.Equal(t, TypeImage, byName["logo"].Type, "logo should be image")

	// Enum detection
	assert.Equal(t, TypeString, byName["priceRange"].Type)
	assert.Contains(t, byName["priceRange"].Options, "budget")
	assert.Contains(t, byName["priceRange"].Options, "mid-range")
	assert.Contains(t, byName["priceRange"].Options, "premium")

	// Array of primitives
	bestFor := byName["bestFor"]
	assert.Equal(t, TypeArray, bestFor.Type)
	require.NotEmpty(t, bestFor.Of)
	assert.Equal(t, "string", bestFor.Of[0].Type)

	// Array of named types
	categoryRatings := byName["categoryRatings"]
	assert.Equal(t, TypeArray, categoryRatings.Type)
	require.NotEmpty(t, categoryRatings.Of)
	assert.Equal(t, "categoryRating", categoryRatings.Of[0].Type)

	// Inline type reference
	seo := byName["seo"]
	assert.Equal(t, FieldType("seoFields"), seo.Type)

	// Polymorphic array (contentSections)
	sections := byName["contentSections"]
	assert.Equal(t, TypeArray, sections.Type)
	assert.Greater(t, len(sections.Of), 2, "should have multiple section types")

	// Nested object
	prosCons := byName["prosCons"]
	assert.Equal(t, TypeObject, prosCons.Type)
	assert.NotEmpty(t, prosCons.Fields)

	// Object types should be resolved
	resolver := v.Resolver()
	assert.NotNil(t, resolver("categoryRating"))
	assert.NotNil(t, resolver("faqItem"))
	assert.NotNil(t, resolver("seoFields"))
}

func TestAddRule_UnknownType(t *testing.T) {
	t.Parallel()
	v, err := NewValidator([]byte(`[{"name":"article","type":"document","attributes":{"name":{"type":"objectAttribute","value":{"type":"string"}}}}]`))
	require.NoError(t, err)

	err = v.AddRule("nonexistent", "name", Rule{Email: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAddRule_UnknownField(t *testing.T) {
	t.Parallel()
	v, err := NewValidator([]byte(`[{"name":"article","type":"document","attributes":{"name":{"type":"objectAttribute","value":{"type":"string"}}}}]`))
	require.NoError(t, err)

	err = v.AddRule("article", "nonexistent", Rule{Email: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAddRule_InvalidRegex(t *testing.T) {
	t.Parallel()
	v, err := NewValidator([]byte(`[{"name":"article","type":"document","attributes":{"name":{"type":"objectAttribute","value":{"type":"string"}}}}]`))
	require.NoError(t, err)

	err = v.AddRule("article", "name", Rule{Regex: "[invalid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile regex")
}

func TestRequire_UnknownType(t *testing.T) {
	t.Parallel()
	v, err := NewValidator([]byte(`[{"name":"article","type":"document","attributes":{"name":{"type":"objectAttribute","value":{"type":"string"}}}}]`))
	require.NoError(t, err)

	assert.False(t, v.Require("nonexistent", "name"))
	assert.True(t, v.Require("article", "name"))
}

// fieldMap creates a map from field name to field for easy test lookups.
func fieldMap(fields []Field) map[string]Field {
	m := make(map[string]Field, len(fields))
	for _, f := range fields {
		m[f.Name] = f
	}
	return m
}
