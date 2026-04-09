# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test -run TestFunctionName ./...

# Lint
golangci-lint run

# Format
gofumpt -w . && goimports -local github.com/lgrote/sanity-validation -w .
```

## Background

This library is the Go equivalent of Sanity Studio's JS validation:

```js
import { validateDocument } from '@sanity/validation'
const result = await validateDocument(doc, schema)
// returns array of { level, message, path }
```

The Go API mirrors this: `Validate(doc, schema, types)` returns `[]Error` with the same `Level`, `Message`, and `Path` semantics. The goal is to run the same validation rules server-side in Go that Sanity Studio enforces client-side in JavaScript.

## Architecture

This is a single-package Go library (`validate`) that validates Sanity CMS documents against a schema, mirroring the rules enforced by Sanity Studio's `@sanity/validation`.

### Two-layer validation pipeline

1. **Structural (Layer 1)**: Implicit invariants — `_key`/`_type` presence, type enforcement, required fields, format checks. Entry point is `validateField` in `field.go`, which dispatches to type-specific validators (`validateString`, `validateArray`, `validateBlock`, etc.).

2. **Rules (Layer 2)**: Explicit rule-based checks (min/max, regex, email, URI, unique, custom validators, etc.). Runs after structural checks via `evaluateRule` in `rule.go`.

### Key abstractions

- **`Document`** → root entity with typed fields and sections
- **`Schema`** → validation blueprint defining `Field` definitions with types, constraints, and `Rule`s
- **`TypeResolver`** (`func(name string) *Schema`) — pluggable function for resolving named types at runtime (used for sections, polymorphic arrays, custom object types)
- **`Error`** — structured result with JSON path, error type (39 constants), got/want, and severity level

### Entry point

`Validate(doc *Document, schema *Schema, types TypeResolver) []Error` in `validate.go` — validates document-level fields, then iterates schema fields and sections.

### File layout by responsibility

| File | Validates |
|------|-----------|
| `validate.go` | Document-level: `_type`, required top-level fields, sections |
| `field.go` | Field type dispatch, primitives (string, number, boolean, date, URL, slug, geopoint), objects, custom types |
| `array.go` | Arrays: bounds, item types, `_key` uniqueness, polymorphic type resolution |
| `block.go` | Portable Text: block structure, spans, markDefs |
| `image.go` | Images, references, asset refs (pre-upload and final formats) |
| `rule.go` | All Rule evaluation (min/max/length, format validators, custom rules) |
| `error.go` | All type definitions: Document, Schema, Field, Rule, Error, constants |

### Testing conventions

- All tests use `t.Parallel()` and `testify/assert`
- Helper `validateOneField(val, f)` wraps a value in a minimal document/schema for isolated field testing
- `validateOneFieldWithResolver` variant adds a `TypeResolver` for named-type tests
