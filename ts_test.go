package validate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestValidator creates a Validator with a minimal schema for testing overlays.
func newTestValidator(typeName string, fieldNames ...string) *Validator {
	var fields []Field
	for _, name := range fieldNames {
		fields = append(fields, Field{Name: name, Type: TypeString})
	}
	return &Validator{
		schemas: map[string]*Schema{
			typeName: {Name: typeName, Fields: fields},
		},
	}
}

func newTestValidatorWithTypes(typeName string, fieldDefs map[string]FieldType) *Validator {
	var fields []Field
	for name, ft := range fieldDefs {
		fields = append(fields, Field{Name: name, Type: ft})
	}
	return &Validator{
		schemas: map[string]*Schema{
			typeName: {Name: typeName, Fields: fields},
		},
	}
}

func TestLoadRules_Required(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "title", "body")
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'title', type: 'string', validation: (Rule) => Rule.required() }),
    defineField({ name: 'body', type: 'string' }),
  ],
})
`)))

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.True(t, byName["title"].Required)
	assert.False(t, byName["body"].Required)
}

func TestLoadRules_MinMax(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "rating")
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'rating', type: 'number', validation: (Rule) => Rule.required().min(0).max(5) }),
  ],
})
`)))

	s := v.Schema("article")
	byName := fieldMap(s.Fields)

	assert.True(t, byName["rating"].Required)
	require.Len(t, byName["rating"].Rules, 1)
	assert.Equal(t, 0, *byName["rating"].Rules[0].Min)
	assert.Equal(t, 5, *byName["rating"].Rules[0].Max)
}

func TestLoadRules_URI(t *testing.T) {
	t.Parallel()

	v := newTestValidator("config", "affiliateUrl")
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'config',
  type: 'document',
  fields: [
    defineField({
      name: 'affiliateUrl',
      type: 'url',
      validation: (Rule) => Rule.uri({ scheme: ['http', 'https'] }),
    }),
  ],
})
`)))

	s := v.Schema("config")
	byName := fieldMap(s.Fields)
	require.Len(t, byName["affiliateUrl"].Rules, 1)
	assert.True(t, byName["affiliateUrl"].Rules[0].URI)
}

func TestLoadRules_TypeRecovery_URL(t *testing.T) {
	t.Parallel()

	v := newTestValidator("brand", "website")
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'brand',
  type: 'document',
  fields: [
    defineField({ name: 'website', title: 'Website', type: 'url', group: 'base' }),
  ],
})
`)))

	s := v.Schema("brand")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeURL, byName["website"].Type)
}

func TestLoadRules_TypeRecovery_Text(t *testing.T) {
	t.Parallel()

	v := newTestValidator("brand", "description")
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'brand',
  type: 'document',
  fields: [
    defineField({ name: 'description', type: 'text', validation: (Rule) => Rule.required() }),
  ],
})
`)))

	s := v.Schema("brand")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeText, byName["description"].Type)
	assert.True(t, byName["description"].Required)
}

func TestLoadRules_TypeRecovery_Date(t *testing.T) {
	t.Parallel()

	v := newTestValidatorWithTypes("seoFields", map[string]FieldType{
		"datePublished": TypeString,
		"dateModified":  TypeString,
		"lastUpdated":   TypeString,
	})
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'seoFields',
  type: 'object',
  fields: [
    defineField({ name: 'datePublished', type: 'date' }),
    defineField({ name: 'dateModified', type: 'date' }),
    defineField({ name: 'lastUpdated', type: 'datetime' }),
  ],
})
`)))

	s := v.Schema("seoFields")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeDate, byName["datePublished"].Type)
	assert.Equal(t, TypeDate, byName["dateModified"].Type)
	assert.Equal(t, TypeDatetime, byName["lastUpdated"].Type)
}

func TestLoadRules_NoOverwriteNonString(t *testing.T) {
	t.Parallel()

	v := newTestValidatorWithTypes("article", map[string]FieldType{
		"rating": TypeNumber,
	})
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'rating', type: 'number', validation: (Rule) => Rule.required() }),
  ],
})
`)))

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeNumber, byName["rating"].Type)
	assert.True(t, byName["rating"].Required)
}

func TestLoadRules_UnknownType(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "title")
	err := v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'nonexistent',
  type: 'document',
  fields: [
    defineField({ name: 'title', type: 'string', validation: (Rule) => Rule.required() }),
  ],
})
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in schema")

	// article schema should be unchanged
	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.False(t, byName["title"].Required)
}

func TestLoadRules_NoValidation(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "title", "body")
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'title', type: 'string' }),
    defineField({ name: 'body', type: 'string' }),
  ],
})
`)))

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.False(t, byName["title"].Required)
	assert.Empty(t, byName["title"].Rules)
}

func TestLoadRules_MultilineField(t *testing.T) {
	t.Parallel()

	v := newTestValidator("brand", "id", "name", "rating")
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'brand',
  title: 'Brand',
  type: 'document',
  fields: [
    defineField({
      name: 'id',
      title: 'Brand ID',
      type: 'string',
      group: 'base',
      validation: (Rule) => Rule.required(),
    }),
    defineField({ name: 'name', title: 'Name', type: 'string', group: 'base', validation: (Rule) => Rule.required() }),
    defineField({
      name: 'rating',
      title: 'Overall Rating',
      type: 'number',
      group: 'base',
      validation: (Rule) => Rule.required().min(0).max(5),
    }),
  ],
})
`)))

	s := v.Schema("brand")
	byName := fieldMap(s.Fields)

	assert.True(t, byName["id"].Required)
	assert.True(t, byName["name"].Required)
	assert.True(t, byName["rating"].Required)
	require.Len(t, byName["rating"].Rules, 1)
	assert.Equal(t, 0, *byName["rating"].Rules[0].Min)
	assert.Equal(t, 5, *byName["rating"].Rules[0].Max)
}

func TestLoadRules_NestedDefineField(t *testing.T) {
	t.Parallel()

	v := newTestValidator("brand", "summary")
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'brand',
  type: 'document',
  fields: [
    defineField({
      name: 'verdict',
      type: 'object',
      fields: [
        defineField({ name: 'summary', type: 'text', validation: (Rule) => Rule.required() }),
      ],
    }),
  ],
})
`)))

	s := v.Schema("brand")
	byName := fieldMap(s.Fields)
	assert.True(t, byName["summary"].Required)
}

func TestLoadRules_BooleanMethods(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "code", "label", "items")
	require.NoError(t, v.LoadRules([]byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'code', type: 'string', validation: (Rule) => Rule.uppercase() }),
    defineField({ name: 'label', type: 'string', validation: (Rule) => Rule.lowercase().warning() }),
    defineField({ name: 'items', type: 'array', validation: (Rule) => Rule.unique() }),
  ],
})
`)))

	s := v.Schema("article")
	byName := fieldMap(s.Fields)

	require.Len(t, byName["code"].Rules, 1)
	assert.True(t, byName["code"].Rules[0].Uppercase)

	require.Len(t, byName["label"].Rules, 1)
	assert.True(t, byName["label"].Rules[0].Lowercase)
	assert.Equal(t, LevelWarning, byName["label"].Rules[0].Level)

	require.Len(t, byName["items"].Rules, 1)
	assert.True(t, byName["items"].Rules[0].Unique)
}

func TestLoadRules_RealSchemas(t *testing.T) {
	t.Parallel()

	schemaJSON := "/Users/lgr/workspace/go/src/github.com/lgrote/websites/dev.pricecomparecar.com/studio/schema.json"

	data, err := os.ReadFile(schemaJSON)
	if err != nil {
		t.Skip("real schema files not available")
	}

	v, err := NewValidator(data)
	require.NoError(t, err)

	// Load individual TS files
	tsFiles := []string{
		"/Users/lgr/workspace/go/src/github.com/lgrote/websites/dev.pricecomparecar.com/studio/schemas/documents/brand.ts",
		"/Users/lgr/workspace/go/src/github.com/lgrote/websites/dev.pricecomparecar.com/studio/schemas/objects/categoryRating.ts",
		"/Users/lgr/workspace/go/src/github.com/lgrote/websites/dev.pricecomparecar.com/studio/schemas/objects/sections/faqSection.ts",
	}
	for _, path := range tsFiles {
		tsData, err := os.ReadFile(path)
		if err != nil {
			t.Skipf("real schema file not available: %s", path)
		}
		require.NoError(t, v.LoadRules(tsData))
	}

	// brand.ts: id, name, rating are required
	brand := v.Schema("brand")
	require.NotNil(t, brand)
	byName := fieldMap(brand.Fields)

	assert.True(t, byName["id"].Required, "brand.id should be required")
	assert.True(t, byName["name"].Required, "brand.name should be required")
	assert.True(t, byName["rating"].Required, "brand.rating should be required")

	// brand.rating has min(0).max(5) rule
	require.NotEmpty(t, byName["rating"].Rules, "brand.rating should have rules")
	assert.Equal(t, 0, *byName["rating"].Rules[0].Min)
	assert.Equal(t, 5, *byName["rating"].Rules[0].Max)

	// Type recovery: brand.website should be TypeURL
	assert.Equal(t, TypeURL, byName["website"].Type, "brand.website should be recovered to TypeURL")

	// Type recovery: brand.description should be TypeText
	assert.Equal(t, TypeText, byName["description"].Type, "brand.description should be recovered to TypeText")

	// categoryRating: rating has required + min/max
	cr := v.Schema("categoryRating")
	require.NotNil(t, cr)
	crByName := fieldMap(cr.Fields)
	assert.True(t, crByName["category"].Required)
	assert.True(t, crByName["rating"].Required)
	require.NotEmpty(t, crByName["rating"].Rules)
	assert.Equal(t, 0, *crByName["rating"].Rules[0].Min)
	assert.Equal(t, 5, *crByName["rating"].Rules[0].Max)

	// faqSection: title required, questions has min(1)
	faq := v.Schema("faqSection")
	require.NotNil(t, faq)
	faqByName := fieldMap(faq.Fields)
	assert.True(t, faqByName["title"].Required)
	require.NotEmpty(t, faqByName["questions"].Rules)
	assert.Equal(t, 1, *faqByName["questions"].Rules[0].Min)
}
