package validate

import (
	"fmt"
	"strings"
)

// ErrorType categorizes a validation error.
type ErrorType string

const (
	ErrNilDocument       ErrorType = "nil_document"
	ErrNilSchema         ErrorType = "nil_schema"
	ErrMissingRequired   ErrorType = "missing_required"
	ErrMissingKey        ErrorType = "missing_key"
	ErrMissingType       ErrorType = "missing_type"
	ErrWrongType         ErrorType = "wrong_type"
	ErrWrongItemType     ErrorType = "wrong_item_type"
	ErrInvalidOption     ErrorType = "invalid_option"
	ErrInvalidFormat     ErrorType = "invalid_format"
	ErrOutOfRange        ErrorType = "out_of_range"
	ErrMinItems          ErrorType = "min_items"
	ErrMaxItems          ErrorType = "max_items"
	ErrDuplicateKey      ErrorType = "duplicate_key"
	ErrRuleMin           ErrorType = "rule_min"
	ErrRuleMax           ErrorType = "rule_max"
	ErrRuleLength        ErrorType = "rule_length"
	ErrRuleRegex         ErrorType = "rule_regex"
	ErrRuleEmail         ErrorType = "rule_email"
	ErrRuleURI           ErrorType = "rule_uri"
	ErrRuleInteger       ErrorType = "rule_integer"
	ErrRulePositive      ErrorType = "rule_positive"
	ErrRuleNegative      ErrorType = "rule_negative"
	ErrRuleUppercase     ErrorType = "rule_uppercase"
	ErrRuleLowercase     ErrorType = "rule_lowercase"
	ErrRuleUnique        ErrorType = "rule_unique"
	ErrRuleAssetRequired ErrorType = "rule_asset_required"
	ErrRuleCustom        ErrorType = "rule_custom"
)

// ErrorLevel indicates the severity of a validation error.
type ErrorLevel string

const (
	LevelError   ErrorLevel = "error"
	LevelWarning ErrorLevel = "warning"
	LevelInfo    ErrorLevel = "info"
)

// FieldType identifies a Sanity schema field type.
type FieldType string

const (
	TypeString    FieldType = "string"
	TypeText      FieldType = "text"
	TypeNumber    FieldType = "number"
	TypeBoolean   FieldType = "boolean"
	TypeDate      FieldType = "date"
	TypeDatetime  FieldType = "datetime"
	TypeURL       FieldType = "url"
	TypeSlug      FieldType = "slug"
	TypeImage     FieldType = "image"
	TypeBlock     FieldType = "block_content"
	TypeArray     FieldType = "array"
	TypeObject    FieldType = "object"
	TypeReference FieldType = "reference"
	TypeGeopoint  FieldType = "geopoint"
)

// Document is a Sanity document to validate.
type Document struct {
	ID          string
	Type        string
	Language    string
	Title       string
	Description string
	Fields      map[string]any
	Sections    []Section
	SectionsKey string
}

// Section is a content section within a document.
type Section struct {
	Type   string
	Key    string
	Fields map[string]any
}

// Schema describes a Sanity document or object type.
type Schema struct {
	Name   string
	Fields []Field
}

// Field describes a single field in a Sanity schema.
type Field struct {
	Name     string
	Type     FieldType
	Required bool
	Of       []ArrayItem // array: allowed item types
	Fields   []Field     // object: nested fields
	Options  []string    // string: allowed enum values
	MinItems *int
	MaxItems *int
	Rules    []Rule // explicit validation rules
}

// ArrayItem describes a type allowed in a Sanity array field.
type ArrayItem struct {
	Type   string  // named type or primitive ("string", "number")
	Fields []Field // inline object definition
}

// TypeResolver looks up named object types by name.
// Returns nil if the type is not found.
type TypeResolver func(name string) *Schema

// Rule is a validation rule attached to a field.
// Rules are evaluated after structural checks pass.
type Rule struct {
	Min           *int         // min length (string/array) or value (number)
	Max           *int         // max length (string/array) or value (number)
	Length        *int         // exact length
	Regex         string       // regex pattern
	Email         bool         // must be valid email
	URI           bool         // must be valid URI
	Integer       bool         // must be whole number
	Positive      bool         // must be > 0
	Negative      bool         // must be < 0
	Uppercase     bool         // must be all uppercase
	Lowercase     bool         // must be all lowercase
	Unique        bool         // array items must be unique (excl _key)
	AssetRequired bool         // image/file must have asset ref
	Custom        []CustomRule // user-defined validators
	Level         ErrorLevel   // LevelError (default), LevelWarning, LevelInfo
}

// CustomRule is a user-defined validation function.
type CustomRule func(value any, ctx RuleContext) *RuleError

// RuleContext provides access to parent and document during custom validation.
type RuleContext struct {
	Path     string
	Parent   map[string]any
	Document *Document
}

// RuleError is returned by custom rules on failure.
type RuleError struct {
	Message string
	Path    string // relative path override (optional)
}

// Error is a single validation error with path and context.
type Error struct {
	Path    string     // JSON path, e.g. "fields.prosCons[2].detail"
	Message string     // human/AI-readable description
	Type    ErrorType  // error category: ErrMissingKey, ErrWrongType, ErrMissingRequired, etc.
	Got     string     // what was found
	Want    string     // what was expected
	Level   ErrorLevel // LevelError (default), LevelWarning, LevelInfo
}

func (e Error) Error() string {
	s := fmt.Sprintf("%s: %s", e.Path, e.Message)
	if e.Got != "" || e.Want != "" {
		s += fmt.Sprintf(". got=%s, want=%s", e.Got, e.Want)
	}
	return s
}

// FormatErrors formats errors for LLM consumption — one line per error.
func FormatErrors(errs []Error) string {
	if len(errs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, e := range errs {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(e.Error())
	}
	return b.String()
}
