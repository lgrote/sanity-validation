package validate

import (
	"os"
	"path/filepath"
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

func TestLoadRulesFromSource_Required(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "title", "body")
	v.LoadRulesFromSource(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'title', type: 'string', validation: (Rule) => Rule.required() }),
    defineField({ name: 'body', type: 'string' }),
  ],
})
`)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.True(t, byName["title"].Required)
	assert.False(t, byName["body"].Required)
}

func TestLoadRulesFromSource_MinMax(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "rating")
	v.LoadRulesFromSource(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'rating', type: 'number', validation: (Rule) => Rule.required().min(0).max(5) }),
  ],
})
`)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)

	assert.True(t, byName["rating"].Required)
	require.Len(t, byName["rating"].Rules, 1)
	assert.Equal(t, 0, *byName["rating"].Rules[0].Min)
	assert.Equal(t, 5, *byName["rating"].Rules[0].Max)
}

func TestLoadRulesFromSource_URI(t *testing.T) {
	t.Parallel()

	v := newTestValidator("config", "affiliateUrl")
	v.LoadRulesFromSource(`
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
`)

	s := v.Schema("config")
	byName := fieldMap(s.Fields)
	require.Len(t, byName["affiliateUrl"].Rules, 1)
	assert.True(t, byName["affiliateUrl"].Rules[0].URI)
}

func TestLoadRulesFromSource_TypeRecovery_URL(t *testing.T) {
	t.Parallel()

	v := newTestValidator("brand", "website")
	v.LoadRulesFromSource(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'brand',
  type: 'document',
  fields: [
    defineField({ name: 'website', title: 'Website', type: 'url', group: 'base' }),
  ],
})
`)

	s := v.Schema("brand")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeURL, byName["website"].Type)
}

func TestLoadRulesFromSource_TypeRecovery_Text(t *testing.T) {
	t.Parallel()

	v := newTestValidator("brand", "description")
	v.LoadRulesFromSource(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'brand',
  type: 'document',
  fields: [
    defineField({ name: 'description', type: 'text', validation: (Rule) => Rule.required() }),
  ],
})
`)

	s := v.Schema("brand")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeText, byName["description"].Type)
	assert.True(t, byName["description"].Required)
}

func TestLoadRulesFromSource_TypeRecovery_Date(t *testing.T) {
	t.Parallel()

	v := newTestValidatorWithTypes("seoFields", map[string]FieldType{
		"datePublished": TypeString,
		"dateModified":  TypeString,
		"lastUpdated":   TypeString,
	})
	v.LoadRulesFromSource(`
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
`)

	s := v.Schema("seoFields")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeDate, byName["datePublished"].Type)
	assert.Equal(t, TypeDate, byName["dateModified"].Type)
	assert.Equal(t, TypeDatetime, byName["lastUpdated"].Type)
}

func TestLoadRulesFromSource_NoOverwriteNonString(t *testing.T) {
	t.Parallel()

	// If the field is already TypeNumber (from schema extract), don't overwrite with TypeString
	v := newTestValidatorWithTypes("article", map[string]FieldType{
		"rating": TypeNumber,
	})
	v.LoadRulesFromSource(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'rating', type: 'number', validation: (Rule) => Rule.required() }),
  ],
})
`)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.Equal(t, TypeNumber, byName["rating"].Type) // unchanged
	assert.True(t, byName["rating"].Required)
}

func TestLoadRulesFromSource_UnknownType(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "title")
	// Type "nonexistent" is not in the validator's schemas
	v.LoadRulesFromSource(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'nonexistent',
  type: 'document',
  fields: [
    defineField({ name: 'title', type: 'string', validation: (Rule) => Rule.required() }),
  ],
})
`)

	// article schema should be unchanged
	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.False(t, byName["title"].Required)
}

func TestLoadRulesFromSource_NoValidation(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "title", "body")
	v.LoadRulesFromSource(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'title', type: 'string' }),
    defineField({ name: 'body', type: 'string' }),
  ],
})
`)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.False(t, byName["title"].Required)
	assert.Empty(t, byName["title"].Rules)
}

func TestLoadRulesFromSource_MultilineField(t *testing.T) {
	t.Parallel()

	v := newTestValidator("brand", "id", "name", "rating")
	v.LoadRulesFromSource(`
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
`)

	s := v.Schema("brand")
	byName := fieldMap(s.Fields)

	assert.True(t, byName["id"].Required)
	assert.True(t, byName["name"].Required)
	assert.True(t, byName["rating"].Required)
	require.Len(t, byName["rating"].Rules, 1)
	assert.Equal(t, 0, *byName["rating"].Rules[0].Min)
	assert.Equal(t, 5, *byName["rating"].Rules[0].Max)
}

func TestLoadRulesFromSource_NestedDefineField(t *testing.T) {
	t.Parallel()

	// defineField inside an object field's fields array
	v := newTestValidator("brand", "summary")
	v.LoadRulesFromSource(`
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
`)

	// The nested field "summary" should be overlaid on the "brand" schema
	// since that's the type name from defineType
	s := v.Schema("brand")
	byName := fieldMap(s.Fields)
	assert.True(t, byName["summary"].Required)
}

func TestLoadRulesFromSource_BooleanMethods(t *testing.T) {
	t.Parallel()

	v := newTestValidator("article", "code", "label", "items")
	v.LoadRulesFromSource(`
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
`)

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

func TestLoadRulesFromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "article.ts"), []byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'title', type: 'string', validation: (Rule) => Rule.required() }),
  ],
})
`), 0o644)
	require.NoError(t, err)

	v := newTestValidator("article", "title")
	err = v.LoadRulesFromFile(filepath.Join(dir, "article.ts"))
	require.NoError(t, err)

	s := v.Schema("article")
	byName := fieldMap(s.Fields)
	assert.True(t, byName["title"].Required)
}

func TestLoadRulesFromDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subdir := filepath.Join(dir, "objects")
	require.NoError(t, os.Mkdir(subdir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "article.ts"), []byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'article',
  type: 'document',
  fields: [
    defineField({ name: 'title', type: 'string', validation: (Rule) => Rule.required() }),
  ],
})
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(subdir, "faqItem.ts"), []byte(`
import { defineField, defineType } from 'sanity'
export default defineType({
  name: 'faqItem',
  type: 'object',
  fields: [
    defineField({ name: 'question', type: 'string', validation: (Rule) => Rule.required() }),
  ],
})
`), 0o644))

	// Non-ts file should be ignored
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte(`module.exports = {}`), 0o644))

	v := &Validator{
		schemas: map[string]*Schema{
			"article": {Name: "article", Fields: []Field{{Name: "title", Type: TypeString}}},
			"faqItem": {Name: "faqItem", Fields: []Field{{Name: "question", Type: TypeString}}},
		},
	}

	err := v.LoadRulesFromDir(dir)
	require.NoError(t, err)

	assert.True(t, fieldMap(v.Schema("article").Fields)["title"].Required)
	assert.True(t, fieldMap(v.Schema("faqItem").Fields)["question"].Required)
}

func TestLoadRulesFromDir_RealSchemas(t *testing.T) {
	t.Parallel()

	schemaDir := "/Users/lgr/workspace/go/src/github.com/lgrote/websites/dev.pricecomparecar.com/studio/schemas"
	schemaJSON := "/Users/lgr/workspace/go/src/github.com/lgrote/websites/dev.pricecomparecar.com/studio/schema.json"

	data, err := os.ReadFile(schemaJSON)
	if err != nil {
		t.Skip("real schema files not available")
	}

	v, err := NewValidator(data)
	require.NoError(t, err)

	err = v.LoadRulesFromDir(schemaDir)
	require.NoError(t, err)

	// brand.ts: id, name, title, description are required
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

	// Type recovery: brand.website should be TypeURL (was TypeString from extract)
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
