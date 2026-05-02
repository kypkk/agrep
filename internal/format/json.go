package format

import (
	"encoding/json"
	"sort"
	"unicode"

	"github.com/kypkk/acode/internal/analyzer"
)

// JSON renders signatures and type declarations as a stable JSON document
// suitable for programmatic consumers (MCP servers, agents, future tools).
// The schema is documented in docs/json-schema.md and is committed to be
// additive across versions — fields may be added but never removed or
// renamed without a major version bump.
//
// Output is pretty-printed with 2-space indentation. Functions and types are
// each sorted internally by (line, name).
func JSON(file, pkg string, sigs []analyzer.Signature, types []analyzer.TypeDecl) string {
	functions := make([]jsonFunction, 0, len(sigs))
	for _, s := range sigs {
		kind := s.Kind
		if kind == "" {
			// Defensive default: callers that construct Signatures without
			// going through the analyzer (e.g., test fixtures) get the
			// natural shape rather than an empty "kind" field.
			kind = "func"
		}
		functions = append(functions, jsonFunction{
			Name:       s.Name,
			Kind:       kind,
			Line:       s.Line,
			Exported:   isExportedJSON(s.Name),
			Receiver:   s.Receiver,
			Parameters: ensureStrings(s.Parameters),
			Returns:    ensureStrings(s.ReturnTypes),
			Doc:        s.DocComment,
		})
	}
	sort.SliceStable(functions, func(i, j int) bool {
		if functions[i].Line != functions[j].Line {
			return functions[i].Line < functions[j].Line
		}
		return functions[i].Name < functions[j].Name
	})

	typeEntries := make([]jsonType, 0, len(types))
	for _, t := range types {
		typeEntries = append(typeEntries, newJSONType(t))
	}
	sort.SliceStable(typeEntries, func(i, j int) bool {
		if typeEntries[i].Line != typeEntries[j].Line {
			return typeEntries[i].Line < typeEntries[j].Line
		}
		return typeEntries[i].Name < typeEntries[j].Name
	})

	doc := jsonDocument{
		File:      file,
		Package:   pkg,
		Functions: functions,
		Types:     typeEntries,
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		// json.Marshal of plain structs / strings / slices cannot fail; if it
		// does, that's a programmer bug worth surfacing rather than swallowing.
		panic(err)
	}
	return string(out)
}

// jsonDocument is the top-level shape. Field order in this struct dictates
// field order in the rendered JSON.
type jsonDocument struct {
	File      string         `json:"file"`
	Package   string         `json:"package"`
	Functions []jsonFunction `json:"functions"`
	Types     []jsonType     `json:"types"`
}

type jsonFunction struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Line       int      `json:"line"`
	Exported   bool     `json:"exported"`
	Receiver   string   `json:"receiver"`
	Parameters []string `json:"parameters"`
	Returns    []string `json:"returns"`
	Doc        string   `json:"doc"`
}

// jsonType is a tagged-union over Kind. The Marshal method below emits the
// correct kind-specific shape: struct → fields, interface → methods,
// alias/named → underlying. The kind-irrelevant fields are never emitted —
// consumers can rely on "kind" to know which extra field to read.
type jsonType struct {
	Name       string
	Kind       string
	Line       int
	Exported   bool
	Fields     []jsonField
	Methods    []jsonMethod
	Underlying string
}

type jsonField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type jsonMethod struct {
	Name       string   `json:"name"`
	Parameters []string `json:"parameters"`
	Returns    []string `json:"returns"`
}

// jsonTypeBase holds the fields every type entry shares; embedded in each
// kind-specific anonymous struct so the wire order stays consistent:
// name, kind, line, exported, then the kind-specific block.
type jsonTypeBase struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Line     int    `json:"line"`
	Exported bool   `json:"exported"`
}

func newJSONType(t analyzer.TypeDecl) jsonType {
	exported := isExportedJSON(t.Name)
	switch t.Kind {
	case "interface":
		return jsonType{
			Name: t.Name, Kind: t.Kind, Line: t.Line, Exported: exported,
			Methods: ensureMethods(t.Methods),
		}
	case "struct":
		return jsonType{
			Name: t.Name, Kind: t.Kind, Line: t.Line, Exported: exported,
			Fields: ensureFields(t.Fields),
		}
	default: // alias, named, or anything else
		return jsonType{
			Name: t.Name, Kind: t.Kind, Line: t.Line, Exported: exported,
			Underlying: t.Underlying,
		}
	}
}

func (t jsonType) MarshalJSON() ([]byte, error) {
	base := jsonTypeBase{Name: t.Name, Kind: t.Kind, Line: t.Line, Exported: t.Exported}
	switch t.Kind {
	case "struct":
		fields := t.Fields
		if fields == nil {
			fields = []jsonField{}
		}
		return json.Marshal(struct {
			jsonTypeBase
			Fields []jsonField `json:"fields"`
		}{base, fields})
	case "interface":
		methods := t.Methods
		if methods == nil {
			methods = []jsonMethod{}
		}
		return json.Marshal(struct {
			jsonTypeBase
			Methods []jsonMethod `json:"methods"`
		}{base, methods})
	default:
		return json.Marshal(struct {
			jsonTypeBase
			Underlying string `json:"underlying"`
		}{base, t.Underlying})
	}
}

// ensureStrings converts a nil slice to a non-nil empty slice so JSON
// marshals it as `[]` rather than `null`. Consumers can iterate without nil
// checks.
func ensureStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func ensureFields(in []analyzer.Field) []jsonField {
	out := make([]jsonField, 0, len(in))
	for _, f := range in {
		out = append(out, jsonField{Name: f.Name, Type: f.Type})
	}
	return out
}

func ensureMethods(in []analyzer.Method) []jsonMethod {
	out := make([]jsonMethod, 0, len(in))
	for _, m := range in {
		out = append(out, jsonMethod{
			Name:       m.Name,
			Parameters: ensureStrings(m.Parameters),
			Returns:    ensureStrings(m.ReturnTypes),
		})
	}
	return out
}

// isExportedJSON mirrors the same exportedness rule used by the CLI filter
// (first rune uppercase). Duplicated here rather than depending on cmd/acode
// to avoid an import cycle.
func isExportedJSON(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}
