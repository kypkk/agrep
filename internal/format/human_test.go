package format

import (
	"strings"
	"testing"

	"github.com/kypkk/acode/internal/analyzer"
)

func TestHuman_Empty(t *testing.T) {
	if got := Human("", "", nil, nil, HumanOptions{}); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestHuman_NoAnsiWhenColorDisabled(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "Hello", Line: 3, Parameters: []string{"x int"}, ReturnTypes: []string{"error"}}}
	got := Human("", "", sigs, nil, HumanOptions{Color: false})
	if strings.Contains(got, "\x1b[") {
		t.Errorf("output contains ANSI escapes when Color=false: %q", got)
	}
}

func TestHuman_AnsiPresentWhenColorEnabled(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "Hello", Line: 3}}
	got := Human("", "", sigs, nil, HumanOptions{Color: true})
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("expected ANSI escapes when Color=true: %q", got)
	}
}

func TestHuman_FunctionContainsKeyParts(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "Hello", Line: 3, Parameters: []string{"x int"}, ReturnTypes: []string{"error"}}}
	got := Human("", "", sigs, nil, HumanOptions{})
	for _, want := range []string{"Hello", "x int", "error", "3"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
}

func TestHuman_DocCommentRendered(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "F", Line: 1, DocComment: "Does the thing.\nReturns nothing."}}
	got := Human("", "", sigs, nil, HumanOptions{})
	if !strings.Contains(got, "Does the thing.") {
		t.Errorf("missing doc first line: %q", got)
	}
	if !strings.Contains(got, "Returns nothing.") {
		t.Errorf("missing doc second line: %q", got)
	}
}

func TestHuman_MethodUsesMethodKeywordAndReceiver(t *testing.T) {
	sigs := []analyzer.Signature{{
		Name: "Parse", Kind: "method", Receiver: "(g *GoParser)", Line: 23,
		Parameters: []string{"src []byte"}, ReturnTypes: []string{"*Tree", "error"},
	}}
	got := Human("", "", sigs, nil, HumanOptions{})
	for _, want := range []string{"method ", "(g *GoParser)", "Parse", "src []byte", "*Tree", "23"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
	// "method" must come first (it's the kind keyword), then the receiver,
	// then the method name. This mirrors the agent format ordering.
	if strings.Index(got, "method ") > strings.Index(got, "(g *GoParser)") {
		t.Errorf("'method' keyword should appear before the receiver: %q", got)
	}
	if strings.Index(got, "(g *GoParser)") > strings.Index(got, "Parse") {
		t.Errorf("receiver should appear before method name: %q", got)
	}
	// Methods must NOT use the `func` keyword (regression guard).
	if strings.Contains(got, "func ") {
		t.Errorf("method line should not contain `func `: %q", got)
	}
}

func TestHuman_FunctionUsesFuncKeywordEvenWithEmptyKind(t *testing.T) {
	// Manually constructed Signatures with the zero Kind value (e.g., test
	// fixtures, future callers) render with the `func` keyword by default.
	// This locks in the same defensive fallback the JSON formatter uses.
	sigs := []analyzer.Signature{{Name: "F", Line: 1}} // Kind == ""
	got := Human("", "", sigs, nil, HumanOptions{})
	if !strings.Contains(got, "func F(") {
		t.Errorf("expected `func F(` in output for empty-Kind signature: %q", got)
	}
	if strings.Contains(got, "method ") {
		t.Errorf("empty-Kind signature must not render as method: %q", got)
	}
}

func TestHuman_StructFieldsRendered(t *testing.T) {
	types := []analyzer.TypeDecl{{
		Name: "Foo", Kind: "struct", Line: 5,
		Fields: []analyzer.Field{{Name: "Name", Type: "string"}, {Name: "Age", Type: "int"}},
	}}
	got := Human("", "", nil, types, HumanOptions{})
	for _, want := range []string{"Foo", "Name", "string", "Age", "int", "5"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
}

func TestHuman_InterfaceMethodsRendered(t *testing.T) {
	types := []analyzer.TypeDecl{{
		Name: "R", Kind: "interface", Line: 10,
		Methods: []analyzer.Method{{Name: "Read", Parameters: []string{"p []byte"}, ReturnTypes: []string{"int", "error"}}},
	}}
	got := Human("", "", nil, types, HumanOptions{})
	for _, want := range []string{"R", "Read", "p []byte", "int", "error", "10"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
}

func TestHuman_AliasAndNamedRendered(t *testing.T) {
	types := []analyzer.TypeDecl{
		{Name: "X", Kind: "alias", Line: 1, Underlying: "string"},
		{Name: "Names", Kind: "named", Line: 2, Underlying: "[]string"},
	}
	got := Human("", "", nil, types, HumanOptions{})
	for _, want := range []string{"X", "= string", "Names", "[]string"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
}

func TestHuman_HeaderShownWithFileAndPackage(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "Hello", Kind: "func", Line: 3}}
	got := Human("src/auth/login.go", "auth", sigs, nil, HumanOptions{})
	for _, want := range []string{"file: src/auth/login.go", "package: auth", "Hello"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
	// Header lines must appear before the entity body.
	if strings.Index(got, "package: auth") > strings.Index(got, "Hello") {
		t.Errorf("header should appear before the body: %q", got)
	}
}

func TestHuman_NoHeaderWhenFileEmpty(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "Hello", Kind: "func", Line: 3}}
	got := Human("", "", sigs, nil, HumanOptions{})
	if strings.Contains(got, "file:") || strings.Contains(got, "package:") {
		t.Errorf("expected no header when file is empty: %q", got)
	}
}

func TestHuman_HeaderShownEvenWithNoEntities(t *testing.T) {
	got := Human("empty.go", "empty", nil, nil, HumanOptions{})
	for _, want := range []string{"file: empty.go", "package: empty"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
}

func TestHuman_OrderedByLine(t *testing.T) {
	sigs := []analyzer.Signature{{Name: "B", Line: 10}, {Name: "A", Line: 5}}
	got := Human("", "", sigs, nil, HumanOptions{})
	if strings.Index(got, "A") > strings.Index(got, "B") {
		t.Errorf("A should appear before B: %q", got)
	}
}
