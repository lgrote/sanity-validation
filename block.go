package validate

import "fmt"

// validateBlock checks a Portable Text (block_content) field value.
// After conversion, block_content is an array of block objects.
func validateBlock(val any, path string, errs *[]Error) {
	arr, ok := val.([]any)
	if !ok {
		// Before conversion, block_content may still be a plain string.
		if _, isStr := val.(string); isStr {
			return // plain text not yet converted — skip block validation
		}
		*errs = append(*errs, Error{
			Path: path, Message: "expected Portable Text array or string",
			Type: ErrWrongType, Got: fmt.Sprintf("%T", val),
			Want: "array of blocks", Level: LevelError,
		})
		return
	}

	for i, item := range arr {
		blockPath := fmt.Sprintf("%s[%d]", path, i)
		block, ok := item.(map[string]any)
		if !ok {
			*errs = append(*errs, Error{
				Path: blockPath, Message: "block must be an object",
				Type: ErrWrongType, Got: fmt.Sprintf("%T", item), Want: "block object", Level: LevelError,
			})
			continue
		}

		// _type
		if t, _ := block["_type"].(string); t == "" {
			*errs = append(*errs, Error{
				Path: blockPath, Message: "block missing _type",
				Type: ErrMissingType, Got: "block without _type", Want: `_type: "block"`, Level: LevelError,
			})
		}

		// _key
		if _, ok := block["_key"].(string); !ok {
			*errs = append(*errs, Error{
				Path: blockPath, Message: "block missing _key",
				Type: ErrMissingKey, Got: "block without _key", Want: "_key string", Level: LevelError,
			})
		}

		// style
		if _, ok := block["style"].(string); !ok {
			*errs = append(*errs, Error{
				Path: blockPath, Message: "block missing style",
				Type: ErrMissingRequired, Got: "block without style", Want: `style string (e.g. "normal")`, Level: LevelError,
			})
		}

		// children
		children, hasChildren := block["children"]
		if !hasChildren {
			*errs = append(*errs, Error{
				Path: blockPath, Message: "block missing children array",
				Type: ErrMissingRequired, Got: "block without children",
				Want: `children: [{_type:"span", _key:..., text:...}]`, Level: LevelError,
			})
		} else if childArr, ok := children.([]any); !ok {
			*errs = append(*errs, Error{
				Path: blockPath + ".children", Message: "children must be an array",
				Type: ErrWrongType, Got: fmt.Sprintf("%T", children), Want: "array of spans", Level: LevelError,
			})
		} else if len(childArr) == 0 {
			*errs = append(*errs, Error{
				Path: blockPath + ".children", Message: "children array is empty",
				Type: ErrMissingRequired, Got: "empty array", Want: "at least one span", Level: LevelError,
			})
		} else {
			for j, child := range childArr {
				validateSpan(child, fmt.Sprintf("%s.children[%d]", blockPath, j), errs)
			}
		}

		// markDefs
		if md, has := block["markDefs"]; has {
			mdArr, ok := md.([]any)
			if !ok {
				*errs = append(*errs, Error{
					Path: blockPath + ".markDefs", Message: "markDefs must be an array",
					Type: ErrWrongType, Got: fmt.Sprintf("%T", md), Want: "array", Level: LevelError,
				})
			} else {
				for j, def := range mdArr {
					validateMarkDef(def, fmt.Sprintf("%s.markDefs[%d]", blockPath, j), errs)
				}
			}
		}
	}
}

// validateSpan checks a single span within a Portable Text block.
func validateSpan(val any, path string, errs *[]Error) {
	span, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "span must be an object",
			Type: ErrWrongType, Got: fmt.Sprintf("%T", val), Want: "span object", Level: LevelError,
		})
		return
	}

	if t, _ := span["_type"].(string); t == "" {
		*errs = append(*errs, Error{
			Path: path, Message: "span missing _type",
			Type: ErrMissingType, Got: "span without _type", Want: `_type: "span"`, Level: LevelError,
		})
	}

	if _, ok := span["_key"].(string); !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "span missing _key",
			Type: ErrMissingKey, Got: "span without _key", Want: "_key string", Level: LevelError,
		})
	}

	if _, ok := span["text"].(string); !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "span missing text",
			Type: ErrMissingRequired, Got: describeValue(span["text"]), Want: "text string", Level: LevelError,
		})
	}
}

// validateMarkDef checks a single mark definition.
func validateMarkDef(val any, path string, errs *[]Error) {
	md, ok := val.(map[string]any)
	if !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "markDef must be an object",
			Type: ErrWrongType, Got: fmt.Sprintf("%T", val), Want: "markDef object", Level: LevelError,
		})
		return
	}

	if _, ok := md["_type"].(string); !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "markDef missing _type",
			Type: ErrMissingType, Got: "markDef without _type", Want: "_type string", Level: LevelError,
		})
	}
	if _, ok := md["_key"].(string); !ok {
		*errs = append(*errs, Error{
			Path: path, Message: "markDef missing _key",
			Type: ErrMissingKey, Got: "markDef without _key", Want: "_key string", Level: LevelError,
		})
	}
}
