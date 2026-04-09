package validate

import "fmt"

// validateImage checks an image field value.
func validateImage(val any, path string, errs *[]Error) {
	m, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "expected image object", Type: ErrWrongType,
			Got: fmt.Sprintf("%T", val), Want: `object with _type: "image"`, Level: LevelError,
		})
		return
	}

	if t, _ := m["_type"].(string); t != "image" {
		*errs = append(*errs, Error{
			Path: path, Message: `image field missing _type: "image"`, Type: ErrMissingType,
			Got: fmt.Sprintf("_type=%q", t), Want: `_type: "image"`, Level: LevelError,
		})
	}

	// If asset is present, validate its structure.
	if asset, has := m["asset"]; has && asset != nil {
		validateAssetRef(asset, path+".asset", errs)
	}
}

// validateReference checks a reference field value.
func validateReference(val any, path string, errs *[]Error) {
	m, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "expected reference object", Type: ErrWrongType,
			Got: fmt.Sprintf("%T", val), Want: `object with _type: "reference" and _ref`, Level: LevelError,
		})
		return
	}

	if t, _ := m["_type"].(string); t != "reference" {
		*errs = append(*errs, Error{
			Path: path, Message: `reference missing _type: "reference"`, Type: ErrMissingType,
			Got: fmt.Sprintf("_type=%q", t), Want: `_type: "reference"`, Level: LevelError,
		})
	}

	if _, ok := m["_ref"].(string); !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "reference missing _ref", Type: ErrMissingRequired,
			Got: describeValue(m["_ref"]), Want: "_ref string", Level: LevelError,
		})
	}
}

// validateAssetRef checks the asset sub-object on an image or file field.
// Accepts both final Sanity format ({_type:"reference", _ref:"..."}) and
// pre-upload format ({url:"...", source:"..."}) since images are uploaded
// and transformed during the upload step.
func validateAssetRef(val any, path string, errs *[]Error) {
	m, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "asset must be an object", Type: ErrWrongType,
			Got: fmt.Sprintf("%T", val), Want: `{_type: "reference", _ref: "..."}`, Level: LevelError,
		})
		return
	}

	// Pre-upload asset: has "url" field with a local path or remote URL.
	// This is valid — the upload step will convert it to a Sanity reference.
	if _, hasURL := m["url"].(string); hasURL {
		return
	}

	// Final Sanity asset reference format.
	if t, _ := m["_type"].(string); t != "reference" {
		*errs = append(*errs, Error{
			Path: path, Message: `asset missing _type: "reference"`, Type: ErrMissingType,
			Got: fmt.Sprintf("_type=%q", t), Want: `_type: "reference"`, Level: LevelError,
		})
	}

	if _, ok := m["_ref"].(string); !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "asset missing _ref", Type: ErrMissingRequired,
			Got: describeValue(m["_ref"]), Want: "_ref string (e.g. image-abc-100x100-jpg)", Level: LevelError,
		})
	}
}
