package validate

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// LoadRules extracts validation rules and Sanity field types from a
// TypeScript schema source (using defineType/defineField from 'sanity')
// and overlays them on matching schemas in this Validator.
//
// Returns an error if the source does not contain a defineType declaration
// or if the type is not found in the Validator's schemas.
func (v *Validator) LoadRules(data []byte) error {
	typeName, fields := parseTS(string(data))
	if typeName == "" {
		return errors.New("no defineType found in source")
	}
	if v.schemas[typeName] == nil {
		return fmt.Errorf("type %q not found in schema", typeName)
	}
	return v.overlayTSFields(typeName, fields)
}

// tsField holds parsed validation info for a single field from a TS schema.
type tsField struct {
	Name     string
	Type     string // Sanity type from TS (url, text, date, etc.)
	Required bool
	Rule     Rule
}

// --- Regex patterns ---

var (
	reTypeName   = regexp.MustCompile(`defineType\(\s*\{\s*name:\s*'([^']+)'`)
	reFieldBlock = regexp.MustCompile(`defineField\(\s*\{`)
	reFieldName  = regexp.MustCompile(`name:\s*'([^']+)'`)
	reFieldType  = regexp.MustCompile(`type:\s*'([^']+)'`)
	reValidation = regexp.MustCompile(`validation:\s*\(\s*Rule\s*\)\s*=>\s*Rule((?:\.[a-zA-Z]+\([^)]*\))*)`)
	reMethodCall = regexp.MustCompile(`\.([a-zA-Z]+)\(([^)]*)\)`)
)

// parseTS extracts the type name and field definitions from a TS schema source.
func parseTS(source string) (string, []tsField) {
	m := reTypeName.FindStringSubmatch(source)
	if m == nil {
		return "", nil
	}
	typeName := m[1]

	blocks := extractFieldBlocks(source)
	var fields []tsField
	for _, block := range blocks {
		f := parseFieldBlock(block)
		if f != nil {
			fields = append(fields, *f)
		}
	}

	return typeName, fields
}

// extractFieldBlocks finds each defineField({...}) block by matching braces.
func extractFieldBlocks(source string) []string {
	var blocks []string
	for _, loc := range reFieldBlock.FindAllStringIndex(source, -1) {
		start := loc[1] // position after "defineField({"
		block := extractBraceBlock(source, start-1)
		if block != "" {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// extractBraceBlock returns the content between balanced {} starting at pos
// (which should point to the opening '{').
// Skips braces inside string literals (single-quoted, double-quoted, backtick).
func extractBraceBlock(source string, pos int) string {
	if pos >= len(source) || source[pos] != '{' {
		return ""
	}
	depth := 0
	for i := pos; i < len(source); i++ {
		ch := source[i]

		// Skip single-line comments.
		if ch == '/' && i+1 < len(source) && source[i+1] == '/' {
			for i < len(source) && source[i] != '\n' {
				i++
			}
			continue
		}
		// Skip block comments.
		if ch == '/' && i+1 < len(source) && source[i+1] == '*' {
			i += 2
			for i+1 < len(source) && (source[i] != '*' || source[i+1] != '/') {
				i++
			}
			i++ // skip closing '/'
			continue
		}

		// Skip string literals.
		if ch == '\'' || ch == '"' || ch == '`' {
			i++
			for i < len(source) && source[i] != ch {
				if source[i] == '\\' {
					i++ // skip escaped character
				}
				i++
			}
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[pos : i+1]
			}
		}
	}
	return ""
}

// parseFieldBlock extracts field info from a single defineField({...}) block.
// Returns nil if no useful info is found.
func parseFieldBlock(block string) *tsField {
	nameMatch := reFieldName.FindStringSubmatch(block)
	if nameMatch == nil {
		return nil
	}

	f := &tsField{Name: nameMatch[1]}

	if typeMatch := reFieldType.FindStringSubmatch(block); typeMatch != nil {
		f.Type = typeMatch[1]
	}

	if valMatch := reValidation.FindStringSubmatch(block); valMatch != nil {
		chain := valMatch[1]
		parseValidationChain(chain, f)
	}

	// Only return if we found validation rules or a recoverable type
	if !f.Required && isEmptyRule(f.Rule) && !isRecoverableType(f.Type) {
		return nil
	}

	return f
}

// parseValidationChain parses a chain like ".required().min(0).max(5)" into tsField.
func parseValidationChain(chain string, f *tsField) {
	for _, m := range reMethodCall.FindAllStringSubmatch(chain, -1) {
		method := m[1]
		arg := strings.TrimSpace(m[2])

		switch method {
		case "required":
			f.Required = true
		case "min":
			if n, err := strconv.Atoi(arg); err == nil {
				f.Rule.Min = &n
			}
		case "max":
			if n, err := strconv.Atoi(arg); err == nil {
				f.Rule.Max = &n
			}
		case "length":
			if n, err := strconv.Atoi(arg); err == nil {
				f.Rule.Length = &n
			}
		case "uri":
			f.Rule.URI = true
		case "email":
			f.Rule.Email = true
		case "integer":
			f.Rule.Integer = true
		case "positive":
			f.Rule.Positive = true
		case "negative":
			f.Rule.Negative = true
		case "uppercase":
			f.Rule.Uppercase = true
		case "lowercase":
			f.Rule.Lowercase = true
		case "unique":
			f.Rule.Unique = true
		case "warning":
			f.Rule.Level = LevelWarning
		case "info":
			f.Rule.Level = LevelInfo
		case "regex":
			// Try to extract pattern from /pattern/ or /pattern/flags
			arg = strings.Trim(arg, " ")
			if len(arg) > 2 && arg[0] == '/' {
				if end := strings.LastIndex(arg, "/"); end > 0 {
					f.Rule.Regex = arg[1:end]
				}
			}
		}
	}
}

// overlayTSFields merges parsed TS field info onto the schema.
func (v *Validator) overlayTSFields(typeName string, fields []tsField) error {
	schema := v.schemas[typeName]
	if schema == nil {
		return nil
	}

	for _, tf := range fields {
		for i := range schema.Fields {
			if schema.Fields[i].Name != tf.Name {
				continue
			}

			if tf.Required {
				schema.Fields[i].Required = true
			}

			if !isEmptyRule(tf.Rule) {
				if tf.Rule.Regex != "" && tf.Rule.compiledRegex == nil {
					var err error
					tf.Rule.compiledRegex, err = regexp.Compile(tf.Rule.Regex)
					if err != nil {
						return fmt.Errorf("field %q: compile regex %q: %w", tf.Name, tf.Rule.Regex, err)
					}
				}
				schema.Fields[i].Rules = append(schema.Fields[i].Rules, tf.Rule)
			}

			if recovered, ok := recoverType(tf.Type); ok && schema.Fields[i].Type == TypeString {
				schema.Fields[i].Type = recovered
			}

			break
		}
	}
	return nil
}

func isEmptyRule(r Rule) bool {
	return r.Min == nil && r.Max == nil && r.Length == nil &&
		r.Regex == "" && !r.Email && !r.URI &&
		!r.Integer && !r.Positive && !r.Negative &&
		!r.Uppercase && !r.Lowercase && !r.Unique &&
		!r.AssetRequired && r.Level == "" && len(r.Custom) == 0
}

func isRecoverableType(t string) bool {
	switch t {
	case "url", "text", "date", "datetime", "slug":
		return true
	}
	return false
}

func recoverType(t string) (FieldType, bool) {
	switch t {
	case "url":
		return TypeURL, true
	case "text":
		return TypeText, true
	case "date":
		return TypeDate, true
	case "datetime":
		return TypeDatetime, true
	case "slug":
		return TypeSlug, true
	}
	return "", false
}
