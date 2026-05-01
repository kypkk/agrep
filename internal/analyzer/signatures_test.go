package analyzer

import (
	"reflect"
	"testing"

	"github.com/kypkk/acode/internal/parser"
)

func parseGo(t *testing.T, src string) (*parser.Tree, []byte) {
	t.Helper()
	bs := []byte(src)
	tree, err := parser.NewGoParser().Parse(bs)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return tree, bs
}

func names(sigs []Signature) []string {
	out := make([]string, len(sigs))
	for i, s := range sigs {
		out[i] = s.Name
	}
	return out
}

func TestExtractSignatures_NoFunctions(t *testing.T) {
	tree, src := parseGo(t, "package x\n\nvar Y = 1\n")
	sigs := ExtractSignatures(tree, src)
	if len(sigs) != 0 {
		t.Fatalf("got %d signatures, want 0: %+v", len(sigs), sigs)
	}
}

func TestExtractSignatures_OneFunction(t *testing.T) {
	tree, src := parseGo(t, "package x\n\nfunc Hello() {}\n")
	sigs := ExtractSignatures(tree, src)
	if len(sigs) != 1 {
		t.Fatalf("got %d signatures, want 1: %+v", len(sigs), sigs)
	}
	if sigs[0].Name != "Hello" {
		t.Errorf("Name = %q, want %q", sigs[0].Name, "Hello")
	}
	if sigs[0].Line != 3 {
		t.Errorf("Line = %d, want 3", sigs[0].Line)
	}
}

func TestExtractSignatures_MultipleFunctions(t *testing.T) {
	src := `package x

func A() {}

func B() {}

func c() {}
`
	tree, bs := parseGo(t, src)
	sigs := ExtractSignatures(tree, bs)
	if len(sigs) != 3 {
		t.Fatalf("got %d signatures, want 3: %+v", len(sigs), sigs)
	}
	got := names(sigs)
	want := []string{"A", "B", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sigs[%d].Name = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExtractSignatures_MethodsWithValueAndPointerReceivers(t *testing.T) {
	src := `package x

type T struct{}

func (t T) ValMethod() {}

func (t *T) PtrMethod() {}
`
	tree, bs := parseGo(t, src)
	sigs := ExtractSignatures(tree, bs)
	if len(sigs) != 2 {
		t.Fatalf("got %d signatures, want 2: %+v", len(sigs), sigs)
	}
	seen := map[string]bool{}
	for _, s := range sigs {
		seen[s.Name] = true
	}
	if !seen["ValMethod"] {
		t.Errorf("missing ValMethod in %v", names(sigs))
	}
	if !seen["PtrMethod"] {
		t.Errorf("missing PtrMethod in %v", names(sigs))
	}
}

func TestExtractSignatures_ParametersAndReturnTypes(t *testing.T) {
	cases := []struct {
		name        string
		src         string
		wantParams  []string
		wantReturns []string
	}{
		{
			name: "zero params, no return",
			src:  "package x\nfunc F() {}\n",
		},
		{
			name:       "single named param",
			src:        "package x\nfunc F(x int) {}\n",
			wantParams: []string{"x int"},
		},
		{
			name:       "multi named params different types",
			src:        "package x\nfunc F(a int, b string) {}\n",
			wantParams: []string{"a int", "b string"},
		},
		{
			name:       "unnamed params",
			src:        "package x\nfunc F(int, string) {}\n",
			wantParams: []string{"int", "string"},
		},
		{
			name:       "variadic",
			src:        "package x\nfunc F(args ...string) {}\n",
			wantParams: []string{"args ...string"},
		},
		{
			name:       "pointer and slice params",
			src:        "package x\nfunc F(p *T, s []int) {}\n",
			wantParams: []string{"p *T", "s []int"},
		},
		{
			name:        "single unnamed return",
			src:         "package x\nfunc F() int { return 0 }\n",
			wantReturns: []string{"int"},
		},
		{
			name:        "multi unnamed return",
			src:         "package x\nfunc F() (int, error) { return 0, nil }\n",
			wantReturns: []string{"int", "error"},
		},
		{
			name:        "named multi return",
			src:         "package x\nfunc F() (x int, err error) { return }\n",
			wantReturns: []string{"x int", "err error"},
		},
		{
			name:        "pointer return",
			src:         "package x\nfunc F() *T { return nil }\n",
			wantReturns: []string{"*T"},
		},
		{
			name:        "slice return",
			src:         "package x\nfunc F() []string { return nil }\n",
			wantReturns: []string{"[]string"},
		},
		{
			name:        "generic type parameters",
			src:         "package x\nfunc F[T any](x T) T { var z T; return z }\n",
			wantParams:  []string{"x T"},
			wantReturns: []string{"T"},
		},
		{
			name:        "method with pointer receiver",
			src:         "package x\ntype T struct{}\nfunc (t *T) F(x int) error { return nil }\n",
			wantParams:  []string{"x int"},
			wantReturns: []string{"error"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tree, src := parseGo(t, tc.src)
			sigs := ExtractSignatures(tree, src)
			if len(sigs) != 1 {
				t.Fatalf("got %d signatures, want 1: %+v", len(sigs), sigs)
			}
			got := sigs[0]
			if !reflect.DeepEqual(got.Parameters, tc.wantParams) {
				t.Errorf("Parameters = %#v, want %#v", got.Parameters, tc.wantParams)
			}
			if !reflect.DeepEqual(got.ReturnTypes, tc.wantReturns) {
				t.Errorf("ReturnTypes = %#v, want %#v", got.ReturnTypes, tc.wantReturns)
			}
		})
	}
}

func TestExtractSignatures_DocComment(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "no comments above function",
			src:  "package x\nfunc F() {}\n",
			want: "",
		},
		{
			name: "single line doc",
			src:  "package x\n// Hello.\nfunc F() {}\n",
			want: "Hello.",
		},
		{
			name: "multi-line doc preserves line breaks",
			src:  "package x\n// First line.\n// Second line.\nfunc F() {}\n",
			want: "First line.\nSecond line.",
		},
		{
			name: "blank line breaks doc association",
			src:  "package x\n// Not a doc comment.\n\nfunc F() {}\n",
			want: "",
		},
		{
			name: "comment without space after slashes",
			src:  "package x\n//tight\nfunc F() {}\n",
			want: "tight",
		},
		{
			name: "comment with extra space preserves indent",
			src:  "package x\n//  indented\nfunc F() {}\n",
			want: " indented",
		},
		{
			name: "unrelated comment plus real doc",
			src:  "package x\n// File header.\n\n// F does X.\nfunc F() {}\n",
			want: "F does X.",
		},
		{
			name: "method with pointer receiver gets doc",
			src:  "package x\ntype T struct{}\n// M does the thing.\nfunc (t *T) M() {}\n",
			want: "M does the thing.",
		},
		{
			name: "trailing comment is not a doc",
			src:  "package x\nfunc F() {} // trailing\n",
			want: "",
		},
		{
			name: "block comment is ignored",
			src:  "package x\n/* block */\nfunc F() {}\n",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tree, src := parseGo(t, tc.src)
			sigs := ExtractSignatures(tree, src)
			if len(sigs) != 1 {
				t.Fatalf("got %d signatures, want 1: %+v", len(sigs), sigs)
			}
			if sigs[0].DocComment != tc.want {
				t.Errorf("DocComment = %q, want %q", sigs[0].DocComment, tc.want)
			}
		})
	}
}

func TestExtractSignatures_DocCommentAcrossMultipleFunctions(t *testing.T) {
	src := `package x

// A is the first.
func A() {}

// B is the second.
func B() {}

func C() {}
`
	tree, bs := parseGo(t, src)
	sigs := ExtractSignatures(tree, bs)
	if len(sigs) != 3 {
		t.Fatalf("got %d signatures, want 3: %+v", len(sigs), sigs)
	}
	want := map[string]string{
		"A": "A is the first.",
		"B": "B is the second.",
		"C": "",
	}
	for _, s := range sigs {
		if got := s.DocComment; got != want[s.Name] {
			t.Errorf("%s.DocComment = %q, want %q", s.Name, got, want[s.Name])
		}
	}
}

func TestExtractSignatures_AnonymousFunctionsExcluded(t *testing.T) {
	src := `package x

var f = func() int { return 1 }

func Outer() {
	g := func() int { return 2 }
	_ = g
}
`
	tree, bs := parseGo(t, src)
	sigs := ExtractSignatures(tree, bs)
	if len(sigs) != 1 {
		t.Fatalf("got %d signatures, want 1 (only Outer): %+v", len(sigs), sigs)
	}
	if sigs[0].Name != "Outer" {
		t.Errorf("Name = %q, want %q", sigs[0].Name, "Outer")
	}
}
