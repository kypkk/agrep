package format

import (
	"strings"
	"testing"

	"github.com/kypkk/acode/internal/analyzer"
)

func TestAgent_Empty(t *testing.T) {
	if got := Agent("", "", nil, nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestAgent_HeaderShownWithFileAndPackage(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "F", Kind: "func", Line: 1}}
	want := "file: src/auth/login.go\npackage: auth\nfunc 1 F()\n"
	if got := Agent("src/auth/login.go", "auth", sigs, nil); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_HeaderShownEvenWithNoEntities(t *testing.T) {
	want := "file: empty.go\npackage: empty\n"
	if got := Agent("empty.go", "empty", nil, nil); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_NoHeaderWhenFileEmpty(t *testing.T) {
	// Back-compat for fixture tests that don't care about the header.
	sigs := []analyzer.Signature{{Name: "F", Kind: "func", Line: 1}}
	want := "func 1 F()\n"
	if got := Agent("", "", sigs, nil); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_FunctionShapes(t *testing.T) {
	cases := []struct {
		name string
		sig  analyzer.Signature
		want string
	}{
		{
			name: "no params no returns",
			sig:  analyzer.Signature{Name: "F", Line: 1},
			want: "func 1 F()\n",
		},
		{
			name: "single param single return",
			sig:  analyzer.Signature{Name: "Hello", Line: 3, Parameters: []string{"x int"}, ReturnTypes: []string{"error"}},
			want: "func 3 Hello(x int) error\n",
		},
		{
			name: "multi param multi return",
			sig:  analyzer.Signature{Name: "F", Line: 5, Parameters: []string{"a int", "b string"}, ReturnTypes: []string{"int", "error"}},
			want: "func 5 F(a int, b string) (int, error)\n",
		},
		{
			name: "named returns",
			sig:  analyzer.Signature{Name: "F", Line: 7, ReturnTypes: []string{"x int", "err error"}},
			want: "func 7 F() (x int, err error)\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Agent("", "", []analyzer.Signature{tc.sig}, nil)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAgent_MethodLineUsesMethodKindAndReceiver(t *testing.T) {
	sigs := []analyzer.Signature{{
		Name: "Parse", Kind: "method", Receiver: "(g *GoParser)", Line: 23,
		Parameters: []string{"src []byte"}, ReturnTypes: []string{"*Tree", "error"},
	}}
	want := "method 23 (g *GoParser) Parse(src []byte) (*Tree, error)\n"
	if got := Agent("", "", sigs, nil); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_FuncLineUnchangedWhenKindFunc(t *testing.T) {
	// Regression guard: setting Kind="func" must not alter the existing
	// function rendering. (Previous tests fed Kind="" which goes through the
	// same path; this asserts the new explicit "func" value also works.)
	sigs := []analyzer.Signature{{
		Name: "Hello", Kind: "func", Line: 3,
		Parameters: []string{"x int"}, ReturnTypes: []string{"error"},
	}}
	want := "func 3 Hello(x int) error\n"
	if got := Agent("", "", sigs, nil); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_EmptyKindDefaultsToFunc(t *testing.T) {
	// Same defensive default as the JSON formatter: a Signature with no Kind
	// renders as `func`, never as `method` or with an empty keyword.
	sigs := []analyzer.Signature{{Name: "F", Line: 1}}
	want := "func 1 F()\n"
	if got := Agent("", "", sigs, nil); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_Struct(t *testing.T) {
	types := []analyzer.TypeDecl{{
		Name: "Foo", Kind: "struct", Line: 5,
		Fields: []analyzer.Field{{Name: "Name", Type: "string"}, {Name: "Age", Type: "int"}},
	}}
	want := "struct 5 Foo {Name string; Age int}\n"
	if got := Agent("", "", nil, types); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_StructWithEmbedded(t *testing.T) {
	types := []analyzer.TypeDecl{{
		Name: "Foo", Kind: "struct", Line: 1,
		Fields: []analyzer.Field{{Type: "io.Reader"}, {Name: "Name", Type: "string"}},
	}}
	want := "struct 1 Foo {io.Reader; Name string}\n"
	if got := Agent("", "", nil, types); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_EmptyStruct(t *testing.T) {
	types := []analyzer.TypeDecl{{Name: "Foo", Kind: "struct", Line: 1}}
	want := "struct 1 Foo {}\n"
	if got := Agent("", "", nil, types); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_Interface(t *testing.T) {
	types := []analyzer.TypeDecl{{
		Name: "R", Kind: "interface", Line: 10,
		Methods: []analyzer.Method{
			{Name: "Read", Parameters: []string{"p []byte"}, ReturnTypes: []string{"int", "error"}},
			{Name: "Close", ReturnTypes: []string{"error"}},
			{Name: "Reset"},
		},
	}}
	want := "interface 10 R {Read(p []byte) (int, error); Close() error; Reset()}\n"
	if got := Agent("", "", nil, types); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_EmptyInterface(t *testing.T) {
	types := []analyzer.TypeDecl{{Name: "Any", Kind: "interface", Line: 1}}
	want := "interface 1 Any {}\n"
	if got := Agent("", "", nil, types); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_Alias(t *testing.T) {
	types := []analyzer.TypeDecl{{Name: "X", Kind: "alias", Line: 1, Underlying: "string"}}
	want := "alias 1 X = string\n"
	if got := Agent("", "", nil, types); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_NamedType(t *testing.T) {
	types := []analyzer.TypeDecl{{Name: "Names", Kind: "named", Line: 1, Underlying: "[]string"}}
	want := "named 1 Names []string\n"
	if got := Agent("", "", nil, types); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_SortedByLineInterleavingFuncsAndTypes(t *testing.T) {
	sigs := []analyzer.Signature{
		{Name: "B", Line: 10},
		{Name: "A", Line: 5},
	}
	types := []analyzer.TypeDecl{
		{Name: "T", Kind: "named", Line: 7, Underlying: "int"},
	}
	want := "func 5 A()\nnamed 7 T int\nfunc 10 B()\n"
	if got := Agent("", "", sigs, types); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_TieBreakingByName(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "B", Line: 1}, {Name: "A", Line: 1}}
	want := "func 1 A()\nfunc 1 B()\n"
	if got := Agent("", "", sigs, nil); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAgent_DeterministicForShuffledInput(t *testing.T) {
	a := []analyzer.Signature{{Name: "A", Line: 1}, {Name: "B", Line: 2}, {Name: "C", Line: 3}}
	b := []analyzer.Signature{{Name: "C", Line: 3}, {Name: "A", Line: 1}, {Name: "B", Line: 2}}
	if x, y := Agent("", "", a, nil), Agent("", "", b, nil); x != y {
		t.Errorf("differ:\n%q\n%q", x, y)
	}
}

func TestAgent_NoAnsiCodes(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "F", Line: 1}}
	got := Agent("", "", sigs, nil)
	if strings.Contains(got, "\x1b[") {
		t.Errorf("agent output must not contain ANSI escapes: %q", got)
	}
}

func TestAgent_Idempotent(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "F", Line: 1, Parameters: []string{"x int"}}}
	types := []analyzer.TypeDecl{{Name: "T", Kind: "struct", Line: 5, Fields: []analyzer.Field{{Name: "X", Type: "int"}}}}
	if Agent("", "", sigs, types) != Agent("", "", sigs, types) {
		t.Error("Agent is not idempotent")
	}
}
