# acode JSON output schema

This document describes the JSON document produced by `acode signatures
<file> --format=json`. The schema is intended for programmatic consumers —
MCP servers, agent skills, downstream tooling.

## Stability commitment

The schema is **additive**. Across versions of acode:

- New fields **may** be added at any level. Consumers MUST ignore unknown
  fields rather than reject them.
- Existing fields **will not** be removed, renamed, or have their type
  changed without a major version bump (a future `v2` schema, signalled in
  release notes).
- The shape of arrays (always `[]`, never `null` for empty) is permanent.

If you build a consumer today, it will keep working against future acode
versions unless you opt into a major bump.

## Top-level shape

```json
{
  "file":      "<path that was passed to acode>",
  "package":   "<Go package name declared in the file>",
  "functions": [ <function entries> ],
  "types":     [ <type entries> ]
}
```

| Field | Type | Notes |
|---|---|---|
| `file` | string | The file path as supplied by the caller. May be relative or absolute depending on how acode was invoked. |
| `package` | string | The Go package name (e.g., `"analyzer"`). Empty string if the source has no `package` clause. |
| `functions` | array | Top-level function and method declarations. Always present, may be empty. |
| `types` | array | Top-level type declarations (struct, interface, alias, named type). Always present, may be empty. |

Both `functions` and `types` are independently sorted by `(line ascending,
name ascending)`.

## Function entry

A "function" entry covers both top-level functions and methods. The `kind`
field discriminates: methods additionally populate `receiver`.

```json
{
  "name":       "Hello",
  "kind":       "func",
  "line":       3,
  "exported":   true,
  "receiver":   "",
  "parameters": ["name string", "times int"],
  "returns":    ["error"],
  "doc":        "Hello greets the world."
}
```

```json
{
  "name":       "Parse",
  "kind":       "method",
  "line":       23,
  "exported":   true,
  "receiver":   "(g *GoParser)",
  "parameters": ["src []byte"],
  "returns":    ["*Tree", "error"],
  "doc":        ""
}
```

| Field | Type | Notes |
|---|---|---|
| `name` | string | The function or method name (no receiver). |
| `kind` | string | `"func"` for top-level functions, `"method"` for methods. |
| `line` | integer | 1-indexed source line where the declaration begins. |
| `exported` | bool | `true` if the first rune of `name` is uppercase. |
| `receiver` | string | The receiver expression as written in source, including parentheses (e.g., `"(g *GoParser)"`, `"(t T)"`, `"(*T)"`). Empty string `""` for `kind: "func"`. Always present. |
| `parameters` | array of strings | One entry per `parameter_declaration` in source order. Multi-name shared-type declarations like `x, y int` stay as a single entry `"x, y int"` for now. Always present, may be `[]`. |
| `returns` | array of strings | One entry per return value. A bare-type return (`func() int`) is a single-element array `["int"]`. A parenthesised list (`func() (int, error)` or `func() (x int, err error)`) is enumerated. Always present, may be `[]`. |
| `doc` | string | Doc comment immediately above the declaration with the `//` prefix (and one optional space) stripped, lines joined by `"\n"`. Empty string if no doc comment. Block comments (`/* */`) are not extracted. |

## Type entry

Type entries are a tagged union over `kind`. Every entry has the common
fields below; the kind-specific block (`fields`, `methods`, or
`underlying`) is **always present** for its kind and **never present** for
other kinds. Consumers can switch on `kind` and access the matching field
without nil-checking.

### Common fields

| Field | Type | Notes |
|---|---|---|
| `name` | string | The type name. |
| `kind` | string | One of `"struct"`, `"interface"`, `"alias"`, `"named"`. |
| `line` | integer | 1-indexed source line. |
| `exported` | bool | First rune uppercase. |

### `kind: "struct"`

```json
{
  "name": "User",
  "kind": "struct",
  "line": 5,
  "exported": true,
  "fields": [
    { "name": "Name",  "type": "string" },
    { "name": "Email", "type": "string" },
    { "name": "",      "type": "io.Reader" }
  ]
}
```

| Field | Type | Notes |
|---|---|---|
| `fields` | array of `{name, type}` | Always present. Empty struct → `[]`. |

Each field entry:

| Field | Type | Notes |
|---|---|---|
| `name` | string | The field name. **Empty string `""`** for embedded fields (e.g., `io.Reader`). |
| `type` | string | The field type as written in source (e.g., `"string"`, `"*User"`, `"[]int"`, `"map[string]int"`). |

Multi-name declarations like `Name, Email string` expand into one entry per
name, sharing the same `type` value. Tags are not yet captured (reserved
for a future schema-additive update).

### `kind: "interface"`

```json
{
  "name": "Reader",
  "kind": "interface",
  "line": 10,
  "exported": true,
  "methods": [
    {
      "name": "Read",
      "parameters": ["p []byte"],
      "returns": ["n int", "err error"]
    },
    {
      "name": "Close",
      "parameters": [],
      "returns": ["error"]
    }
  ]
}
```

| Field | Type | Notes |
|---|---|---|
| `methods` | array of method objects | Always present. Empty interface → `[]`. |

Each method entry has `name`, `parameters`, and `returns` with the same
shape as in function entries.

Type-element constraints (`~int | string`, embedded interface lines) are
not yet emitted in v0 — they will be added under a separate field name
when supported (additive change).

### `kind: "alias"` and `kind: "named"`

```json
{
  "name": "Names",
  "kind": "alias",
  "line": 20,
  "exported": true,
  "underlying": "[]string"
}
```

```json
{
  "name": "Counter",
  "kind": "named",
  "line": 21,
  "exported": true,
  "underlying": "int"
}
```

| Field | Type | Notes |
|---|---|---|
| `underlying` | string | The underlying type expression as written in source. |

`alias` is `type Name = T` (true alias — same type as `T`). `named` is
`type Name T` (a new distinct type with `T` as its underlying type).

## Empty document

A file with no top-level functions or types still produces a valid
document:

```json
{
  "file": "empty.go",
  "package": "empty",
  "functions": [],
  "types": []
}
```

## Filtering

By default `acode signatures` only emits exported (capital-prefixed)
top-level entries. Pass `--all` to include unexported. The members of an
included type (struct fields, interface methods) are always shown
regardless of `--all`. The `exported` flag on each entry tells you the
visibility unconditionally — you do not need to inspect `name` yourself.

## Determinism

For the same input the JSON output is **byte-identical** across runs and
across machines. Field order within objects is fixed by acode's emitter;
arrays are sorted by `(line, name)`.

## Example

Source:

```go
package demo

// Hello greets you.
func Hello() error { return nil }

type User struct {
    Name string
}
```

Output of `acode signatures demo.go --format=json`:

```json
{
  "file": "demo.go",
  "package": "demo",
  "functions": [
    {
      "name": "Hello",
      "line": 4,
      "exported": true,
      "parameters": [],
      "returns": ["error"],
      "doc": "Hello greets you."
    }
  ],
  "types": [
    {
      "name": "User",
      "kind": "struct",
      "line": 6,
      "exported": true,
      "fields": [
        { "name": "Name", "type": "string" }
      ]
    }
  ]
}
```
