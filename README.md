# sanity-validation

A Go library for validating [Sanity](https://www.sanity.io/) documents against a schema -- the same structural and rule-based checks that Sanity Studio enforces client-side via [`@sanity/validation`](https://github.com/sanity-io/sanity/tree/next/packages/%40sanity/validation), but running server-side in Go.

## Motivation

Sanity Studio validates documents in the browser before they are saved:

```js
import { validateDocument } from '@sanity/validation'

const result = await validateDocument(doc, schema)
// [{ level: 'error', message: 'Required', path: ['title'] }, ...]
```

This library brings equivalent validation to Go, so you can run the same checks in API servers, data pipelines, or migration scripts without depending on a JS runtime.

## Install

```bash
go get github.com/lgrote/sanity-validation
```

## Usage

```go
package main

import (
	"fmt"

	validate "github.com/lgrote/sanity-validation"
)

func main() {
	doc := &validate.Document{
		Type:  "article",
		Title: "Hello World",
		Fields: map[string]any{
			"slug": map[string]any{"current": "hello-world"},
			"body": []any{
				map[string]any{
					"_type": "block",
					"_key":  "abc",
					"style": "normal",
					"children": []any{
						map[string]any{
							"_type": "span",
							"_key":  "s1",
							"text":  "Hello!",
						},
					},
				},
			},
		},
	}

	schema := &validate.Schema{
		Name: "article",
		Fields: []validate.Field{
			{Name: "title", Type: validate.TypeString, Required: true},
			{Name: "slug", Type: validate.TypeSlug, Required: true},
			{Name: "body", Type: validate.TypeBlock, Required: true},
		},
	}

	errs := validate.Validate(doc, schema, nil)
	if len(errs) == 0 {
		fmt.Println("Document is valid")
		return
	}
	for _, e := range errs {
		fmt.Printf("[%s] %s: %s\n", e.Level, e.Path, e.Message)
	}
}
```

## What it validates

### Layer 1 -- Structural checks

Implicit invariants that every Sanity document must satisfy:

- `_type` and `_key` presence on documents, sections, and array items
- Field type enforcement (string, number, boolean, date, datetime, URL, slug, image, reference, geopoint, object, array)
- Required field checks
- Portable Text (block content) structure: blocks, spans, markDefs
- Image and reference structure, including pre-upload and final asset formats
- Array item type discrimination (primitive vs. object), duplicate key detection, min/max item bounds
- Enum option validation for string fields

### Layer 2 -- Rule-based checks

Explicit `Rule` constraints attached to schema fields:

| Rule | Applies to | Check |
|------|-----------|-------|
| `Min` / `Max` | string, number, array | Length or value bounds |
| `Length` | string, array | Exact length |
| `Regex` | string | Pattern match |
| `Email` | string | Valid email address |
| `URI` | string | Valid URI with scheme |
| `Integer` | number | Whole number |
| `Positive` / `Negative` | number | Sign check |
| `Uppercase` / `Lowercase` | string | Case check |
| `Unique` | array | No duplicate items (excluding `_key`) |
| `AssetRequired` | image | Asset reference must be present |
| `Custom` | any | User-defined `func(value any, ctx RuleContext) *RuleError` |

Each rule can set a `Level` (`error`, `warning`, or `info`) to control severity.

## Type resolution

For documents with sections or polymorphic arrays, pass a `TypeResolver` function to look up named types at validation time:

```go
types := func(name string) *validate.Schema {
	switch name {
	case "hero":
		return &validate.Schema{
			Name: "hero",
			Fields: []validate.Field{
				{Name: "heading", Type: validate.TypeString, Required: true},
				{Name: "image", Type: validate.TypeImage, Required: true},
			},
		}
	default:
		return nil
	}
}

errs := validate.Validate(doc, schema, types)
```

## Built with AI

This project was built with the help of [Claude Code](https://claude.ai/code) and [Gemini](https://gemini.google.com/). The AI assistants helped with initial code generation, test coverage, and iterating on the validation logic to match Sanity Studio's behavior.

## License

MIT
