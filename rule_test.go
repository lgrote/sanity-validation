package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- String length counts runes, not bytes ---

func TestRule_Min_String_Unicode(t *testing.T) {
	t.Parallel()
	// "日本語" is 3 runes (9 bytes) — min(3) should pass.
	errs := validateOneField("日本語", Field{Name: "s", Type: TypeString, Rules: []Rule{{Min: new(3)}}})
	assert.Empty(t, errs)
}

func TestRule_Max_String_Unicode(t *testing.T) {
	t.Parallel()
	// "日本語!" is 4 runes — max(3) should fail.
	errs := validateOneField("日本語!", Field{Name: "s", Type: TypeString, Rules: []Rule{{Max: new(3)}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleMax, errs[0].Type)
	assert.Contains(t, errs[0].Message, "length 4")
}

// --- String min/max ---

func TestRule_Min_String_TooShort(t *testing.T) {
	t.Parallel()
	errs := validateOneField("ab", Field{Name: "s", Type: TypeString, Rules: []Rule{{Min: new(3)}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleMin, errs[0].Type)
}

func TestRule_Min_String_OK(t *testing.T) {
	t.Parallel()
	errs := validateOneField("abc", Field{Name: "s", Type: TypeString, Rules: []Rule{{Min: new(3)}}})
	assert.Empty(t, errs)
}

func TestRule_Max_String_TooLong(t *testing.T) {
	t.Parallel()
	errs := validateOneField("abcd", Field{Name: "s", Type: TypeString, Rules: []Rule{{Max: new(3)}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleMax, errs[0].Type)
}

func TestRule_Max_String_OK(t *testing.T) {
	t.Parallel()
	errs := validateOneField("abc", Field{Name: "s", Type: TypeString, Rules: []Rule{{Max: new(3)}}})
	assert.Empty(t, errs)
}

// --- Number min/max ---

func TestRule_Min_Number_Below(t *testing.T) {
	t.Parallel()
	errs := validateOneField(1.0, Field{Name: "n", Type: TypeNumber, Rules: []Rule{{Min: new(5)}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleMin, errs[0].Type)
}

func TestRule_Max_Number_Above(t *testing.T) {
	t.Parallel()
	errs := validateOneField(10.0, Field{Name: "n", Type: TypeNumber, Rules: []Rule{{Max: new(5)}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleMax, errs[0].Type)
}

// --- Array min/max ---

func TestRule_Min_Array(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{"a"},
		Field{Name: "a", Type: TypeArray, Of: []ArrayItem{{Type: "string"}}, Rules: []Rule{{Min: new(2)}}},
	)
	assert.NotEmpty(t, errs)
	hasMin := false
	for _, e := range errs {
		if e.Type == ErrRuleMin {
			hasMin = true
		}
	}
	assert.True(t, hasMin, "expected rule_min error")
}

func TestRule_Max_Array(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{"a", "b", "c"},
		Field{Name: "a", Type: TypeArray, Of: []ArrayItem{{Type: "string"}}, Rules: []Rule{{Max: new(2)}}},
	)
	assert.NotEmpty(t, errs)
	hasMax := false
	for _, e := range errs {
		if e.Type == ErrRuleMax {
			hasMax = true
		}
	}
	assert.True(t, hasMax, "expected rule_max error")
}

// --- Length ---

func TestRule_Length_String_Match(t *testing.T) {
	t.Parallel()
	errs := validateOneField("abc", Field{Name: "s", Type: TypeString, Rules: []Rule{{Length: new(3)}}})
	assert.Empty(t, errs)
}

func TestRule_Length_String_Mismatch(t *testing.T) {
	t.Parallel()
	errs := validateOneField("ab", Field{Name: "s", Type: TypeString, Rules: []Rule{{Length: new(3)}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleLength, errs[0].Type)
}

// --- Regex ---

func TestRule_Regex_Match(t *testing.T) {
	t.Parallel()
	errs := validateOneField("abc123", Field{Name: "s", Type: TypeString, Rules: []Rule{{Regex: `^[a-z]+[0-9]+$`}}})
	assert.Empty(t, errs)
}

func TestRule_Regex_NoMatch(t *testing.T) {
	t.Parallel()
	errs := validateOneField("ABC", Field{Name: "s", Type: TypeString, Rules: []Rule{{Regex: `^[a-z]+[0-9]+$`}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleRegex, errs[0].Type)
}

// --- Email ---

func TestRule_Email_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("user@example.com", Field{Name: "e", Type: TypeString, Rules: []Rule{{Email: true}}})
	assert.Empty(t, errs)
}

func TestRule_Email_Invalid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("not-an-email", Field{Name: "e", Type: TypeString, Rules: []Rule{{Email: true}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleEmail, errs[0].Type)
}

// --- URI ---

func TestRule_URI_Valid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("https://example.com", Field{Name: "u", Type: TypeString, Rules: []Rule{{URI: true}}})
	assert.Empty(t, errs)
}

func TestRule_URI_Invalid(t *testing.T) {
	t.Parallel()
	errs := validateOneField("not a uri", Field{Name: "u", Type: TypeString, Rules: []Rule{{URI: true}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleURI, errs[0].Type)
}

// --- Integer ---

func TestRule_Integer_Whole(t *testing.T) {
	t.Parallel()
	errs := validateOneField(5.0, Field{Name: "n", Type: TypeNumber, Rules: []Rule{{Integer: true}}})
	assert.Empty(t, errs)
}

func TestRule_Integer_Fractional(t *testing.T) {
	t.Parallel()
	errs := validateOneField(5.5, Field{Name: "n", Type: TypeNumber, Rules: []Rule{{Integer: true}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleInteger, errs[0].Type)
}

// --- Positive ---

func TestRule_Positive_OK(t *testing.T) {
	t.Parallel()
	errs := validateOneField(1.0, Field{Name: "n", Type: TypeNumber, Rules: []Rule{{Positive: true}}})
	assert.Empty(t, errs)
}

func TestRule_Positive_Zero(t *testing.T) {
	t.Parallel()
	errs := validateOneField(0.0, Field{Name: "n", Type: TypeNumber, Rules: []Rule{{Positive: true}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRulePositive, errs[0].Type)
}

func TestRule_Positive_Negative(t *testing.T) {
	t.Parallel()
	errs := validateOneField(-1.0, Field{Name: "n", Type: TypeNumber, Rules: []Rule{{Positive: true}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRulePositive, errs[0].Type)
}

// --- Negative ---

func TestRule_Negative_OK(t *testing.T) {
	t.Parallel()
	errs := validateOneField(-1.0, Field{Name: "n", Type: TypeNumber, Rules: []Rule{{Negative: true}}})
	assert.Empty(t, errs)
}

func TestRule_Negative_Zero(t *testing.T) {
	t.Parallel()
	errs := validateOneField(0.0, Field{Name: "n", Type: TypeNumber, Rules: []Rule{{Negative: true}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleNegative, errs[0].Type)
}

// --- Uppercase ---

func TestRule_Uppercase_OK(t *testing.T) {
	t.Parallel()
	errs := validateOneField("ABC", Field{Name: "s", Type: TypeString, Rules: []Rule{{Uppercase: true}}})
	assert.Empty(t, errs)
}

func TestRule_Uppercase_Mixed(t *testing.T) {
	t.Parallel()
	errs := validateOneField("Abc", Field{Name: "s", Type: TypeString, Rules: []Rule{{Uppercase: true}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleUppercase, errs[0].Type)
}

// --- Lowercase ---

func TestRule_Lowercase_OK(t *testing.T) {
	t.Parallel()
	errs := validateOneField("abc", Field{Name: "s", Type: TypeString, Rules: []Rule{{Lowercase: true}}})
	assert.Empty(t, errs)
}

func TestRule_Lowercase_Mixed(t *testing.T) {
	t.Parallel()
	errs := validateOneField("Abc", Field{Name: "s", Type: TypeString, Rules: []Rule{{Lowercase: true}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleLowercase, errs[0].Type)
}

// --- Unique ---

func TestRule_Unique_AllUnique(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{
			map[string]any{"_key": "k1", "title": "A"},
			map[string]any{"_key": "k2", "title": "B"},
		},
		Field{Name: "a", Type: TypeArray, Rules: []Rule{{Unique: true}}},
	)
	// Filter only rule_unique errors (there may be structural errors for missing _type).
	var uniqueErrs []Error
	for _, e := range errs {
		if e.Type == ErrRuleUnique {
			uniqueErrs = append(uniqueErrs, e)
		}
	}
	assert.Empty(t, uniqueErrs)
}

func TestRule_Unique_Duplicates(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		[]any{
			map[string]any{"_key": "k1", "title": "A"},
			map[string]any{"_key": "k2", "title": "A"},
		},
		Field{Name: "a", Type: TypeArray, Rules: []Rule{{Unique: true}}},
	)
	hasUnique := false
	for _, e := range errs {
		if e.Type == ErrRuleUnique {
			hasUnique = true
		}
	}
	assert.True(t, hasUnique, "expected rule_unique error")
}

// --- AssetRequired ---

func TestRule_AssetRequired_WithAsset(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{
			"_type": "image",
			"asset": map[string]any{"_type": "reference", "_ref": "image-abc"},
		},
		Field{Name: "img", Type: TypeImage, Rules: []Rule{{AssetRequired: true}}},
	)
	assert.Empty(t, errs)
}

func TestRule_AssetRequired_WithoutAsset(t *testing.T) {
	t.Parallel()
	errs := validateOneField(
		map[string]any{"_type": "image"},
		Field{Name: "img", Type: TypeImage, Rules: []Rule{{AssetRequired: true}}},
	)
	assert.NotEmpty(t, errs)
	hasAsset := false
	for _, e := range errs {
		if e.Type == ErrRuleAssetRequired {
			hasAsset = true
		}
	}
	assert.True(t, hasAsset, "expected rule_asset_required error")
}

// --- Custom rules ---

func TestRule_Custom_Pass(t *testing.T) {
	t.Parallel()
	custom := func(value any, ctx RuleContext) *RuleError {
		return nil
	}
	errs := validateOneField("ok", Field{Name: "s", Type: TypeString, Rules: []Rule{{Custom: []CustomRule{custom}}}})
	assert.Empty(t, errs)
}

func TestRule_Custom_Fail(t *testing.T) {
	t.Parallel()
	custom := func(value any, ctx RuleContext) *RuleError {
		return &RuleError{Message: "bad"}
	}
	errs := validateOneField("ok", Field{Name: "s", Type: TypeString, Rules: []Rule{{Custom: []CustomRule{custom}}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, ErrRuleCustom, errs[0].Type)
	assert.Contains(t, errs[0].Message, "bad")
}

func TestRule_Custom_PathOverride(t *testing.T) {
	t.Parallel()
	custom := func(value any, ctx RuleContext) *RuleError {
		return &RuleError{Message: "nested bad", Path: "nested.field"}
	}
	errs := validateOneField("ok", Field{Name: "s", Type: TypeString, Rules: []Rule{{Custom: []CustomRule{custom}}}})
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Path, "nested.field")
}

// --- Level ---

func TestRule_Level_Warning(t *testing.T) {
	t.Parallel()
	errs := validateOneField("ab", Field{Name: "s", Type: TypeString, Rules: []Rule{{Min: new(3), Level: LevelWarning}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, LevelWarning, errs[0].Level)
}

func TestRule_Level_Error(t *testing.T) {
	t.Parallel()
	errs := validateOneField("ab", Field{Name: "s", Type: TypeString, Rules: []Rule{{Min: new(3)}}})
	assert.NotEmpty(t, errs)
	assert.Equal(t, LevelError, errs[0].Level)
}

// --- Multiple rules ---

func TestRule_Multiple_AllChecked(t *testing.T) {
	t.Parallel()
	errs := validateOneField("ab", Field{
		Name: "s", Type: TypeString,
		Rules: []Rule{
			{Min: new(5)},
			{Uppercase: true},
		},
	})
	hasMin := false
	hasUpper := false
	for _, e := range errs {
		if e.Type == ErrRuleMin {
			hasMin = true
		}
		if e.Type == ErrRuleUppercase {
			hasUpper = true
		}
	}
	assert.True(t, hasMin, "expected rule_min error")
	assert.True(t, hasUpper, "expected rule_uppercase error")
}
