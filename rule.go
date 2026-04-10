package validate

import (
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// evaluateRule evaluates a single Rule against a field value.
func evaluateRule(val any, r Rule, f Field, path string, _ TypeResolver, parent map[string]any, doc *Document, errs *[]Error) {
	level := r.Level
	if level == "" {
		level = LevelError
	}

	// Min/Max — behavior depends on field type.
	switch f.Type { //nolint:exhaustive // only string, text, number, array have min/max semantics
	case TypeString, TypeText:
		s, _ := val.(string)
		n := utf8.RuneCountInString(s)
		if r.Min != nil && n < *r.Min {
			*errs = append(*errs, Error{
				Path: path, Message: fmt.Sprintf("string length %d is below minimum %d", n, *r.Min),
				Type: ErrRuleMin, Got: fmt.Sprintf("length %d", n), Want: fmt.Sprintf(">= %d chars", *r.Min), Level: level,
			})
		}
		if r.Max != nil && n > *r.Max {
			*errs = append(*errs, Error{
				Path: path, Message: fmt.Sprintf("string length %d exceeds maximum %d", n, *r.Max),
				Type: ErrRuleMax, Got: fmt.Sprintf("length %d", n), Want: fmt.Sprintf("<= %d chars", *r.Max), Level: level,
			})
		}
		if r.Length != nil && n != *r.Length {
			*errs = append(*errs, Error{
				Path: path, Message: fmt.Sprintf("string length %d, expected exactly %d", n, *r.Length),
				Type: ErrRuleLength, Got: fmt.Sprintf("length %d", n), Want: fmt.Sprintf("exactly %d chars", *r.Length), Level: level,
			})
		}
	case TypeNumber:
		n, ok := toFloat(val)
		if ok {
			if r.Min != nil && n < float64(*r.Min) {
				*errs = append(*errs, Error{
					Path: path, Message: fmt.Sprintf("value %g is below minimum %d", n, *r.Min),
					Type: ErrRuleMin, Got: fmt.Sprintf("%g", n), Want: fmt.Sprintf(">= %d", *r.Min), Level: level,
				})
			}
			if r.Max != nil && n > float64(*r.Max) {
				*errs = append(*errs, Error{
					Path: path, Message: fmt.Sprintf("value %g exceeds maximum %d", n, *r.Max),
					Type: ErrRuleMax, Got: fmt.Sprintf("%g", n), Want: fmt.Sprintf("<= %d", *r.Max), Level: level,
				})
			}
		}
	case TypeArray:
		arr, _ := val.([]any)
		if r.Min != nil && len(arr) < *r.Min {
			*errs = append(*errs, Error{
				Path: path, Message: fmt.Sprintf("array has %d items, minimum is %d", len(arr), *r.Min),
				Type: ErrRuleMin, Got: fmt.Sprintf("%d items", len(arr)), Want: fmt.Sprintf(">= %d items", *r.Min), Level: level,
			})
		}
		if r.Max != nil && len(arr) > *r.Max {
			*errs = append(*errs, Error{
				Path: path, Message: fmt.Sprintf("array has %d items, maximum is %d", len(arr), *r.Max),
				Type: ErrRuleMax, Got: fmt.Sprintf("%d items", len(arr)), Want: fmt.Sprintf("<= %d items", *r.Max), Level: level,
			})
		}
		if r.Length != nil && len(arr) != *r.Length {
			*errs = append(*errs, Error{
				Path: path, Message: fmt.Sprintf("array has %d items, expected exactly %d", len(arr), *r.Length),
				Type: ErrRuleLength, Got: fmt.Sprintf("%d items", len(arr)), Want: fmt.Sprintf("exactly %d items", *r.Length), Level: level,
			})
		}
	}

	// Regex.
	if r.Regex != "" {
		s, _ := val.(string)
		re := r.CompiledRegex
		if re == nil {
			var err error
			re, err = regexp.Compile(r.Regex)
			if err != nil {
				re = nil
			}
		}
		if re != nil && !re.MatchString(s) {
			*errs = append(*errs, Error{
				Path: path, Message: "value does not match pattern " + r.Regex,
				Type: ErrRuleRegex, Got: describeValue(val), Want: "match " + r.Regex, Level: level,
			})
		}
	}

	// Email.
	if r.Email {
		s, _ := val.(string)
		if _, err := mail.ParseAddress(s); err != nil {
			*errs = append(*errs, Error{
				Path: path, Message: "invalid email address",
				Type: ErrRuleEmail, Got: s, Want: "valid email", Level: level,
			})
		}
	}

	// URI.
	if r.URI {
		s, _ := val.(string)
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" {
			*errs = append(*errs, Error{
				Path: path, Message: "invalid URI",
				Type: ErrRuleURI, Got: s, Want: "valid URI with scheme", Level: level,
			})
		}
	}

	// Integer.
	if r.Integer {
		n, ok := toFloat(val)
		if ok && n != math.Trunc(n) {
			*errs = append(*errs, Error{
				Path: path, Message: "expected integer, got fractional number",
				Type: ErrRuleInteger, Got: fmt.Sprintf("%g", n), Want: "whole number", Level: level,
			})
		}
	}

	// Positive.
	if r.Positive {
		n, ok := toFloat(val)
		if ok && n <= 0 {
			*errs = append(*errs, Error{
				Path: path, Message: "expected positive number",
				Type: ErrRulePositive, Got: fmt.Sprintf("%g", n), Want: "> 0", Level: level,
			})
		}
	}

	// Negative.
	if r.Negative {
		n, ok := toFloat(val)
		if ok && n >= 0 {
			*errs = append(*errs, Error{
				Path: path, Message: "expected negative number",
				Type: ErrRuleNegative, Got: fmt.Sprintf("%g", n), Want: "< 0", Level: level,
			})
		}
	}

	// Uppercase.
	if r.Uppercase {
		s, _ := val.(string)
		if s != "" && s != strings.ToUpper(s) {
			*errs = append(*errs, Error{
				Path: path, Message: "expected all uppercase",
				Type: ErrRuleUppercase, Got: s, Want: strings.ToUpper(s), Level: level,
			})
		}
	}

	// Lowercase.
	if r.Lowercase {
		s, _ := val.(string)
		if s != "" && s != strings.ToLower(s) {
			*errs = append(*errs, Error{
				Path: path, Message: "expected all lowercase",
				Type: ErrRuleLowercase, Got: s, Want: strings.ToLower(s), Level: level,
			})
		}
	}

	// Unique (arrays only).
	if r.Unique {
		arr, _ := val.([]any)
		for i := range arr {
			for j := i + 1; j < len(arr); j++ {
				if deepEqualExcludeKey(arr[i], arr[j]) {
					*errs = append(*errs, Error{
						Path: fmt.Sprintf("%s[%d]", path, j), Message: fmt.Sprintf("duplicate of item [%d]", i),
						Type: ErrRuleUnique, Got: describeValue(arr[j]), Want: "unique items", Level: level,
					})
				}
			}
		}
	}

	// AssetRequired (images/files).
	if r.AssetRequired {
		m, _ := val.(map[string]any)
		if m != nil {
			asset, has := m["asset"]
			if !has || asset == nil {
				*errs = append(*errs, Error{
					Path: path, Message: "asset is required",
					Type: ErrRuleAssetRequired, Got: "image without asset", Want: "image with asset reference", Level: level,
				})
			}
		}
	}

	// Custom rules.
	for _, custom := range r.Custom {
		ctx := RuleContext{
			Path:     path,
			Parent:   parent,
			Document: doc,
		}
		if ruleErr := custom(val, ctx); ruleErr != nil {
			errPath := path
			if ruleErr.Path != "" {
				errPath = path + "." + ruleErr.Path
			}
			*errs = append(*errs, Error{
				Path: errPath, Message: ruleErr.Message,
				Type: ErrRuleCustom, Level: level,
			})
		}
	}
}
