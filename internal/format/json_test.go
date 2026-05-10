package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kypkk/agrep/internal/analyzer"
)

func TestJSON_EmptyShape(t *testing.T) {
	got := JSON("path/to/foo.go", "foo", nil, nil)
	want := `{
  "file": "path/to/foo.go",
  "package": "foo",
  "functions": [],
  "types": []
}`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestJSON_OutputIsValidJSON(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "F", Line: 1, Parameters: []string{"x int"}, ReturnTypes: []string{"error"}, DocComment: "Does it."}}
	types := []analyzer.TypeDecl{
		{Name: "S", Kind: "struct", Line: 5, Fields: []analyzer.Field{{Name: "X", Type: "int"}}},
		{Name: "I", Kind: "interface", Line: 10, Methods: []analyzer.Method{{Name: "Do"}}},
		{Name: "A", Kind: "alias", Line: 20, Underlying: "string"},
		{Name: "N", Kind: "named", Line: 25, Underlying: "int"},
	}
	got := JSON("f.go", "p", sigs, types)
	var any interface{}
	if err := json.Unmarshal([]byte(got), &any); err != nil {
		t.Fatalf("invalid JSON: %v\n---\n%s", err, got)
	}
}

func TestJSON_Function(t *testing.T) {
	sigs := []analyzer.Signature{{
		Name: "Hello", Kind: "func", Line: 3,
		Parameters:  []string{"x int", "y string"},
		ReturnTypes: []string{"int", "error"},
		DocComment:  "Hello does X.\nIt returns Y.",
	}}
	got := JSON("f.go", "p", sigs, nil)
	want := `{
  "file": "f.go",
  "package": "p",
  "functions": [
    {
      "name": "Hello",
      "kind": "func",
      "line": 3,
      "exported": true,
      "receiver": "",
      "parameters": [
        "x int",
        "y string"
      ],
      "returns": [
        "int",
        "error"
      ],
      "doc": "Hello does X.\nIt returns Y."
    }
  ],
  "types": []
}`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestJSON_Method(t *testing.T) {
	sigs := []analyzer.Signature{{
		Name: "Parse", Kind: "method", Receiver: "(g *GoParser)", Line: 23,
		Parameters:  []string{"src []byte"},
		ReturnTypes: []string{"*Tree", "error"},
	}}
	got := JSON("f.go", "p", sigs, nil)
	want := `{
  "file": "f.go",
  "package": "p",
  "functions": [
    {
      "name": "Parse",
      "kind": "method",
      "line": 23,
      "exported": true,
      "receiver": "(g *GoParser)",
      "parameters": [
        "src []byte"
      ],
      "returns": [
        "*Tree",
        "error"
      ],
      "doc": ""
    }
  ],
  "types": []
}`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestJSON_FunctionDefaultsKindToFuncWhenEmpty(t *testing.T) {
	// Signatures constructed with the zero value (no Kind set) render as
	// kind=func — keeps callers from accidentally producing JSON with an
	// empty kind string.
	sigs := []analyzer.Signature{{Name: "F", Line: 1}}
	got := JSON("f.go", "p", sigs, nil)
	if !strings.Contains(got, `"kind": "func"`) {
		t.Errorf("expected kind=func default, got:\n%s", got)
	}
}

func TestJSON_FunctionEmptyArrays(t *testing.T) {
	// A function with no params and no returns must still have parameters: []
	// and returns: [] — never null and never absent.
	sigs := []analyzer.Signature{{Name: "F", Line: 1}}
	got := JSON("f.go", "p", sigs, nil)
	if !strings.Contains(got, `"parameters": []`) {
		t.Errorf("expected `\"parameters\": []`, got:\n%s", got)
	}
	if !strings.Contains(got, `"returns": []`) {
		t.Errorf("expected `\"returns\": []`, got:\n%s", got)
	}
}

func TestJSON_FunctionExported(t *testing.T) {
	sigs := []analyzer.Signature{
		{Name: "Public", Line: 1},
		{Name: "private", Line: 2},
	}
	got := JSON("f.go", "p", sigs, nil)
	var doc struct {
		Functions []struct {
			Name     string `json:"name"`
			Exported bool   `json:"exported"`
		} `json:"functions"`
	}
	if err := json.Unmarshal([]byte(got), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Functions) != 2 {
		t.Fatalf("want 2 functions, got %d", len(doc.Functions))
	}
	if !doc.Functions[0].Exported {
		t.Errorf("Public should be exported")
	}
	if doc.Functions[1].Exported {
		t.Errorf("private should not be exported")
	}
}

func TestJSON_StructHasFieldsArrayNotMethodsNorUnderlying(t *testing.T) {
	types := []analyzer.TypeDecl{{
		Name: "Foo", Kind: "struct", Line: 5,
		Fields: []analyzer.Field{
			{Name: "Name", Type: "string"},
			{Name: "", Type: "io.Reader"},
		},
	}}
	got := JSON("f.go", "p", nil, types)
	want := `{
  "file": "f.go",
  "package": "p",
  "functions": [],
  "types": [
    {
      "name": "Foo",
      "kind": "struct",
      "line": 5,
      "exported": true,
      "fields": [
        {
          "name": "Name",
          "type": "string"
        },
        {
          "name": "",
          "type": "io.Reader"
        }
      ]
    }
  ]
}`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestJSON_EmptyStructHasEmptyFieldsArray(t *testing.T) {
	types := []analyzer.TypeDecl{{Name: "Foo", Kind: "struct", Line: 1}}
	got := JSON("f.go", "p", nil, types)
	if !strings.Contains(got, `"fields": []`) {
		t.Errorf("empty struct must have fields: [], got:\n%s", got)
	}
	if strings.Contains(got, `"methods"`) || strings.Contains(got, `"underlying"`) {
		t.Errorf("struct must not have methods or underlying, got:\n%s", got)
	}
}

func TestJSON_Interface(t *testing.T) {
	types := []analyzer.TypeDecl{{
		Name: "R", Kind: "interface", Line: 10,
		Methods: []analyzer.Method{
			{Name: "Read", Parameters: []string{"p []byte"}, ReturnTypes: []string{"int", "error"}},
			{Name: "Close"},
		},
	}}
	got := JSON("f.go", "p", nil, types)
	want := `{
  "file": "f.go",
  "package": "p",
  "functions": [],
  "types": [
    {
      "name": "R",
      "kind": "interface",
      "line": 10,
      "exported": true,
      "methods": [
        {
          "name": "Read",
          "parameters": [
            "p []byte"
          ],
          "returns": [
            "int",
            "error"
          ]
        },
        {
          "name": "Close",
          "parameters": [],
          "returns": []
        }
      ]
    }
  ]
}`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestJSON_EmptyInterface(t *testing.T) {
	types := []analyzer.TypeDecl{{Name: "Any", Kind: "interface", Line: 1}}
	got := JSON("f.go", "p", nil, types)
	if !strings.Contains(got, `"methods": []`) {
		t.Errorf("empty interface must have methods: [], got:\n%s", got)
	}
	if strings.Contains(got, `"fields"`) || strings.Contains(got, `"underlying"`) {
		t.Errorf("interface must not have fields or underlying, got:\n%s", got)
	}
}

func TestJSON_Alias(t *testing.T) {
	types := []analyzer.TypeDecl{{Name: "X", Kind: "alias", Line: 20, Underlying: "string"}}
	got := JSON("f.go", "p", nil, types)
	want := `{
  "file": "f.go",
  "package": "p",
  "functions": [],
  "types": [
    {
      "name": "X",
      "kind": "alias",
      "line": 20,
      "exported": true,
      "underlying": "string"
    }
  ]
}`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, `"fields"`) || strings.Contains(got, `"methods"`) {
		t.Errorf("alias must not have fields or methods")
	}
}

func TestJSON_NamedType(t *testing.T) {
	types := []analyzer.TypeDecl{{Name: "Names", Kind: "named", Line: 1, Underlying: "[]string"}}
	got := JSON("f.go", "p", nil, types)
	if !strings.Contains(got, `"kind": "named"`) {
		t.Errorf("missing kind=named: %s", got)
	}
	if !strings.Contains(got, `"underlying": "[]string"`) {
		t.Errorf("missing underlying: %s", got)
	}
}

func TestJSON_SortedByLineThenName(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "B", Line: 10}, {Name: "A", Line: 5}}
	types := []analyzer.TypeDecl{{Name: "T", Kind: "named", Line: 7, Underlying: "int"}}
	got := JSON("f.go", "p", sigs, types)
	var doc struct {
		Functions []struct {
			Name string `json:"name"`
			Line int    `json:"line"`
		} `json:"functions"`
		Types []struct {
			Name string `json:"name"`
			Line int    `json:"line"`
		} `json:"types"`
	}
	if err := json.Unmarshal([]byte(got), &doc); err != nil {
		t.Fatal(err)
	}
	// Functions and types are kept in separate arrays per the schema.
	// Within each array, entries are sorted by (line, name).
	if doc.Functions[0].Name != "A" || doc.Functions[1].Name != "B" {
		t.Errorf("functions not sorted by line: %+v", doc.Functions)
	}
	if doc.Types[0].Name != "T" {
		t.Errorf("types[0] = %+v", doc.Types[0])
	}
}

func TestJSON_TieBreakingByName(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "B", Line: 1}, {Name: "A", Line: 1}}
	got := JSON("f.go", "p", sigs, nil)
	idxA := strings.Index(got, `"name": "A"`)
	idxB := strings.Index(got, `"name": "B"`)
	if idxA < 0 || idxB < 0 || idxA > idxB {
		t.Errorf("A should appear before B for line tie:\n%s", got)
	}
}

func TestJSON_Idempotent(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "F", Line: 1, Parameters: []string{"x int"}}}
	types := []analyzer.TypeDecl{{Name: "T", Kind: "struct", Line: 5, Fields: []analyzer.Field{{Name: "X", Type: "int"}}}}
	a := JSON("f.go", "p", sigs, types)
	b := JSON("f.go", "p", sigs, types)
	if a != b {
		t.Error("JSON output is not idempotent")
	}
}
