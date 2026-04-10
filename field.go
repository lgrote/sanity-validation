package validate

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var (
	dateRegex     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	datetimeRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
)

// validateField checks a single field value against its schema definition.
// parent is the containing object (for rule context). doc is the root document.
func validateField(val any, f Field, path string, types TypeResolver, parent map[string]any, doc *Document, errs *[]Error) {
	// Required check.
	if f.Required && isEmpty(val) {
		*errs = append(*errs, Error{
			Path:    path,
			Message: fmt.Sprintf("required field %q is empty or missing", f.Name),
			Type:    ErrMissingRequired,
			Got:     describeValue(val),
			Want:    "non-empty value",
			Level:   LevelError,
		})
		return
	}

	if val == nil {
		return
	}

	// Type-specific structural checks (Layer 1).
	errsBefore := len(*errs)
	switch f.Type {
	case TypeString, TypeText:
		validateString(val, f, path, errs)
	case TypeNumber:
		validateNumber(val, f, path, errs)
	case TypeBoolean:
		validateBoolean(val, path, errs)
	case TypeDate:
		validateDate(val, path, errs)
	case TypeDatetime:
		validateDatetime(val, path, errs)
	case TypeURL:
		validateURL(val, path, errs)
	case TypeSlug:
		validateSlug(val, path, errs)
	case TypeImage:
		validateImage(val, path, errs)
	case TypeBlock:
		validateBlock(val, path, errs)
	case TypeArray:
		validateArray(val, f, path, types, parent, doc, errs)
	case TypeObject:
		validateObject(val, f, path, types, doc, errs)
	case TypeReference:
		validateReference(val, path, errs)
	case TypeGeopoint:
		validateGeopoint(val, path, errs)
	default:
		// Custom type — resolve via TypeResolver.
		validateCustomType(val, f, path, types, doc, errs)
	}

	// Rule-based checks (Layer 2) — only run if structural checks passed.
	if len(*errs) == errsBefore {
		for _, r := range f.Rules {
			evaluateRule(val, r, f, path, parent, doc, errs)
		}
	}
}

func validateString(val any, f Field, path string, errs *[]Error) {
	s, ok := val.(string)
	if !ok {
		*errs = append(*errs, Error{
			Path:    path,
			Message: "expected string",
			Type:    ErrWrongType,
			Got:     fmt.Sprintf("%T", val),
			Want:    "string",
			Level:   LevelError,
		})
		return
	}
	// Enum check.
	if len(f.Options) > 0 && s != "" {
		found := slices.Contains(f.Options, s)
		if !found {
			*errs = append(*errs, Error{
				Path:    path,
				Message: fmt.Sprintf("value %q not in allowed options", s),
				Type:    ErrInvalidOption,
				Got:     s,
				Want:    fmt.Sprintf("one of %v", f.Options),
				Level:   LevelError,
			})
		}
	}
}

func validateNumber(val any, _ Field, path string, errs *[]Error) {
	switch val.(type) {
	case float64, float32, int, int64, int32:
		// valid
	default:
		*errs = append(*errs, Error{
			Path:    path,
			Message: "expected number",
			Type:    ErrWrongType,
			Got:     fmt.Sprintf("%T", val),
			Want:    "number",
			Level:   LevelError,
		})
	}
}

func validateBoolean(val any, path string, errs *[]Error) {
	if _, ok := val.(bool); !ok {
		*errs = append(*errs, Error{
			Path:    path,
			Message: "expected boolean",
			Type:    ErrWrongType,
			Got:     fmt.Sprintf("%T", val),
			Want:    "bool",
			Level:   LevelError,
		})
	}
}

func validateDate(val any, path string, errs *[]Error) {
	s, ok := val.(string)
	if !ok {
		*errs = append(*errs, Error{
			Path:    path,
			Message: "expected date string",
			Type:    ErrWrongType,
			Got:     fmt.Sprintf("%T", val),
			Want:    "string (YYYY-MM-DD)",
			Level:   LevelError,
		})
		return
	}
	if !dateRegex.MatchString(s) {
		*errs = append(*errs, Error{
			Path:    path,
			Message: "invalid date format",
			Type:    ErrInvalidFormat,
			Got:     s,
			Want:    "YYYY-MM-DD",
			Level:   LevelError,
		})
	}
}

func validateDatetime(val any, path string, errs *[]Error) {
	s, ok := val.(string)
	if !ok {
		*errs = append(*errs, Error{
			Path:    path,
			Message: "expected datetime string",
			Type:    ErrWrongType,
			Got:     fmt.Sprintf("%T", val),
			Want:    "string (ISO 8601)",
			Level:   LevelError,
		})
		return
	}
	if !datetimeRegex.MatchString(s) {
		*errs = append(*errs, Error{
			Path:    path,
			Message: "invalid datetime format",
			Type:    ErrInvalidFormat,
			Got:     s,
			Want:    "YYYY-MM-DDTHH:MM:SS",
			Level:   LevelError,
		})
	}
}

func validateURL(val any, path string, errs *[]Error) {
	s, ok := val.(string)
	if !ok {
		*errs = append(*errs, Error{
			Path:    path,
			Message: "expected URL string",
			Type:    ErrWrongType,
			Got:     fmt.Sprintf("%T", val),
			Want:    "string (URL)",
			Level:   LevelError,
		})
		return
	}
	if s != "" {
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" {
			*errs = append(*errs, Error{
				Path:    path,
				Message: "invalid URL",
				Type:    ErrInvalidFormat,
				Got:     s,
				Want:    "valid URL with scheme",
				Level:   LevelError,
			})
		}
	}
}

func validateSlug(val any, path string, errs *[]Error) {
	m, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, Error{
			Path:    path,
			Message: "expected slug object",
			Type:    ErrWrongType,
			Got:     fmt.Sprintf("%T", val),
			Want:    `object with "current" string field`,
			Level:   LevelError,
		})
		return
	}
	current, ok := m["current"]
	if !ok {
		*errs = append(*errs, Error{
			Path:    path + ".current",
			Message: `slug missing "current" field`,
			Type:    ErrMissingRequired,
			Got:     "slug without current",
			Want:    `slug with "current" string`,
			Level:   LevelError,
		})
		return
	}
	if _, ok := current.(string); !ok {
		*errs = append(*errs, Error{
			Path:    path + ".current",
			Message: `slug "current" must be a string`,
			Type:    ErrWrongType,
			Got:     fmt.Sprintf("%T", current),
			Want:    "string",
			Level:   LevelError,
		})
	}
}

func validateGeopoint(val any, path string, errs *[]Error) {
	m, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, Error{
			Path:    path,
			Message: "expected geopoint object",
			Type:    ErrWrongType,
			Got:     fmt.Sprintf("%T", val),
			Want:    "object with lat and lng",
			Level:   LevelError,
		})
		return
	}
	lat, hasLat := toFloat(m["lat"])
	lng, hasLng := toFloat(m["lng"])
	if !hasLat {
		*errs = append(*errs, Error{
			Path: path + ".lat", Message: "missing or invalid lat",
			Type: ErrMissingRequired, Got: describeValue(m["lat"]), Want: "number (-90 to 90)", Level: LevelError,
		})
	} else if lat < -90 || lat > 90 {
		*errs = append(*errs, Error{
			Path: path + ".lat", Message: "lat out of range",
			Type: ErrOutOfRange, Got: fmt.Sprintf("%g", lat), Want: "-90 to 90", Level: LevelError,
		})
	}
	if !hasLng {
		*errs = append(*errs, Error{
			Path: path + ".lng", Message: "missing or invalid lng",
			Type: ErrMissingRequired, Got: describeValue(m["lng"]), Want: "number (-180 to 180)", Level: LevelError,
		})
	} else if lng < -180 || lng > 180 {
		*errs = append(*errs, Error{
			Path: path + ".lng", Message: "lng out of range",
			Type: ErrOutOfRange, Got: fmt.Sprintf("%g", lng), Want: "-180 to 180", Level: LevelError,
		})
	}
}

func validateObject(val any, f Field, path string, types TypeResolver, doc *Document, errs *[]Error) {
	m, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "expected object", Type: ErrWrongType,
			Got: fmt.Sprintf("%T", val), Want: "object", Level: LevelError,
		})
		return
	}
	for _, nested := range f.Fields {
		nVal := m[nested.Name]
		validateField(nVal, nested, path+"."+nested.Name, types, m, doc, errs)
	}
}

func validateCustomType(val any, f Field, path string, types TypeResolver, doc *Document, errs *[]Error) {
	if types == nil {
		return
	}
	schema := types(string(f.Type))
	if schema == nil {
		return // unknown type — can't validate
	}
	m, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, Error{
			Path: path, Message: fmt.Sprintf("expected object (%s)", f.Type), Type: ErrWrongType,
			Got: fmt.Sprintf("%T", val), Want: fmt.Sprintf("object (%s)", f.Type), Level: LevelError,
		})
		return
	}
	for _, nested := range schema.Fields {
		nVal := m[nested.Name]
		validateField(nVal, nested, path+"."+nested.Name, types, m, doc, errs)
	}
}

// isEmpty checks if a value is nil or empty (empty string, empty array, empty map).
func isEmpty(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}

// toFloat extracts a float64 from a numeric value.
func toFloat(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	}
	return 0, false
}

// describeValue returns a short description of a value for error messages.
func describeValue(val any) string {
	if val == nil {
		return "nil"
	}
	switch v := val.(type) {
	case string:
		if len(v) > 40 {
			return fmt.Sprintf("string %q...", v[:40])
		}
		return fmt.Sprintf("string %q", v)
	case float64:
		if v == math.Trunc(v) {
			return fmt.Sprintf("number %d", int64(v))
		}
		return fmt.Sprintf("number %g", v)
	case bool:
		return fmt.Sprintf("bool %v", v)
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		return fmt.Sprintf("object {%s}", strings.Join(keys, ", "))
	case []any:
		return fmt.Sprintf("array (len=%d)", len(v))
	}
	return fmt.Sprintf("%T", val)
}
