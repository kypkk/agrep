# agrep

[![CI](https://github.com/kypkk/acode/actions/workflows/ci.yml/badge.svg)](https://github.com/kypkk/acode/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/kypkk/acode.svg)](https://pkg.go.dev/github.com/kypkk/acode)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kypkk/acode)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> Token-efficient code recon for AI agents. A scout you run before `read`.

`agrep` is a CLI that extracts function signatures, type declarations, and
doc comments from source files, so an AI agent can understand what a file
exposes without reading every byte of it.

## Why

Agents exploring an unfamiliar codebase usually don't have a "find" problem
— `grep` and `ast-grep` already solve that well. They have a "look inside"
problem: to understand what a 4 KB file does, an agent reads the whole
thing into context, which costs about 1,000 tokens for one mid-sized file
and stacks up fast across a survey.

But the agent rarely needs the bodies. It needs the surface: what does
this file export? What are the signatures? Are there doc comments?

agrep renders that surface in roughly **10% of the source size**:

| File | Source | Agent format | Reduction |
|---|---:|---:|---:|
| `internal/analyzer/types.go` | 4,763 B | 575 B | **8.3×** |

Run agrep first to scope what's interesting. Run `read` only on the file
that actually matters.

## Install

```bash
go install github.com/kypkk/acode/cmd/agrep@latest
```

> agrep uses a tree-sitter binding that requires cgo, so a working C
> compiler must be available at install time (Apple clang, gcc, or MSVC).

## Usage

agrep ships two subcommands:

- [`agrep signatures`](#agrep-signatures) — read the structural surface of one file
- [`agrep find`](#agrep-find--cross-file-structural-search) — search across files for declarations matching structural predicates

### `agrep signatures`

```
agrep signatures <file> [--format=human|agent|json] [--all]
```

| Flag | Default | Effect |
|-----|---|---|
| `--format` | `human` | Output format. `human` is for terminals, `agent` is for piping into another agent or tool, `json` is for programmatic consumers. |
| `--all` | `false` | Include unexported (lowercase-prefix) symbols. By default only exported names appear. Members of an *included* type (struct fields, interface methods) are always shown. |

Run `agrep --help` or `agrep signatures --help` for the full reference.

### Human format

```
$ agrep signatures internal/analyzer/types.go --all
struct TypeDecl  # line 14
  Name       string
  Kind       string
  Line       int
  Fields     []Field
  Methods    []Method
  Underlying string

struct Field  # line 26
  Name string
  Type string

// ExtractTypes walks the top level of a Go parse tree and returns one
// TypeDecl per type_spec or type_alias inside any type_declaration. Grouped
// declarations (`type ( A ...; B ...; )`) flatten into separate TypeDecls.
func ExtractTypes(tree *parser.Tree, src []byte) []TypeDecl  # line 42
```

Auto-coloured when stdout is a TTY, plain when piped.

### Agent format

```
$ agrep signatures internal/analyzer/types.go --format=agent --all
struct 14 TypeDecl {Name string; Kind string; Line int; Fields []Field; Methods []Method; Underlying string}
struct 26 Field {Name string; Type string}
struct 33 Method {Name string; Parameters []string; ReturnTypes []string}
func 42 ExtractTypes(tree *parser.Tree, src []byte) []TypeDecl
```

Dense, deterministic, one entity per line. Sorted by `(line, name)` so
the same input always produces byte-identical output. No ANSI codes.

### JSON format

```
$ agrep signatures internal/format/json.go --format=json | jq '.functions[].name'
"JSON"
```

Structured document with a stable, additive schema documented in
[docs/json-schema.md](docs/json-schema.md). Designed for MCP servers,
agent skills, and downstream tooling.

### `agrep find` — cross-file structural search

`signatures` reads one file. `find` answers questions across many — the
ones grep can't reach because they need an AST.

```
agrep find [flags]
```

| Flag | Effect |
|---|---|
| `--has-method <name>` | Types with a method named X |
| `--implements <I>` | Types whose method-name set covers an interface I (name-only, in-scope) |
| `--returns <substr>` | Functions/methods whose return list contains the substring |
| `--takes <substr>` | Functions/methods whose parameter list contains the substring |
| `--has-receiver <T>` | Methods on a specific receiver type |
| `--kind <K>` | Filter by `func`, `method`, `struct`, `interface`, `alias`, `named` |
| `--in <path>` | Directory or file to search (default `.`) |
| `--limit N` / `--all` | Pagination (default 20, `--all` shows everything) |

Flags compose with **AND**. Output groups by *type* when the query is type-shaped
(`--has-method`, `--implements`), flat-lists otherwise.

```
$ agrep find --has-method=Parse --in=.
1 types with method Parse:

GoParser               internal/parser/golang.go:11
  func (g *GoParser) Parse(src []byte) (*Tree, error) :23

1 results
```

```
$ agrep find --returns=error --in=internal/finder/
2 functions return error:

internal/finder/scanner.go:20   func ScanFile(path string) (FileResult, error)
internal/finder/walker.go:14   func Walk(root string) ([]string, error)

2 results
```

When a query returns nothing, agrep proposes nearby names from the corpus:

```
$ agrep find --has-method=Bogus --in=internal/finder/
No results.

Searched 12 files in internal/finder/ for types with method "Bogus".

Did you mean:
  - method=Apply
  - method=Suggest
  - method=Walk
```

## What's supported

agrep v0 is **Go-only**. The codebase is structured behind a
language-agnostic `parser.Parser` interface so additional languages plug
in without touching the analyzer or formatters — but no other languages
ship today.

The `signatures` subcommand extracts:

- Top-level **functions** and **methods** (the method name; the receiver expression is reserved for a future field)
- **Struct** types (fields, including multi-name and embedded)
- **Interface** types (method signatures)
- Type **aliases** (`type X = Y`) and **named** types (`type X Y`)
- **Doc comments** (`//` line comments immediately above an entity)

Not yet captured (planned, additive when added):
- Struct field tags
- Block-comment doc (`/* */`)
- Constants and package-level variables
- Generic type parameters on type declarations (functions are covered)
- Interface type-element constraints (`~int | string`)

## How it works

agrep parses source with [tree-sitter](https://tree-sitter.github.io/)
and walks the AST. Tree-sitter is fast, error-recovering, and
language-agnostic — the same engine handles dozens of languages with
small per-language grammar packages.

The pipeline:

```
file.go ──► parser.GoParser ──► analyzer.ExtractSignatures
                              │
                              └─► analyzer.ExtractTypes
                                       │
                                       └─► format.{Human,Agent,JSON}
```

Each stage lives in its own `internal/` package and is independently
testable.

## Project status

Early. Single language, single subcommand, no tagged release yet. The
JSON schema is committed to be **additive across versions** (see
[docs/json-schema.md](docs/json-schema.md) for the stability promise);
everything else may shift before v1.

## Star History

<a href="https://www.star-history.com/?repos=kypkk%2Facode&type=date&legend=bottom-right">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=kypkk/acode&type=date&theme=dark&legend=bottom-right" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=kypkk/acode&type=date&legend=bottom-right" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=kypkk/acode&type=date&legend=bottom-right" />
 </picture>
</a>
