package finder

import (
	"testing"

	"github.com/kypkk/agrep/internal/analyzer"
)

// sigNames pulls a flat list of names out of Match results for assertions.
// Returns the type name when the match is a type, the signature name when
// it's a func/method.
func sigNames(matches []Match) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		switch {
		case m.Sig != nil:
			out = append(out, m.Sig.Name)
		case m.Type != nil:
			out = append(out, m.Type.Name)
		}
	}
	return out
}

func TestFilter_Kind_Func(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Sigs: []analyzer.Signature{
			{Name: "F", Kind: "func", Line: 1},
			{Name: "M", Kind: "method", Receiver: "(t *T)", Line: 2},
		},
		Types: []analyzer.TypeDecl{{Name: "T", Kind: "struct", Line: 5}},
	}})
	got := Filter{Kind: "func"}.Apply(c)
	if len(got) != 1 || got[0].Sig == nil || got[0].Sig.Name != "F" {
		t.Errorf("got %+v, want one func F", got)
	}
}

func TestFilter_Kind_Struct(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Types: []analyzer.TypeDecl{
			{Name: "Foo", Kind: "struct", Line: 1},
			{Name: "Bar", Kind: "interface", Line: 2},
		},
	}})
	got := Filter{Kind: "struct"}.Apply(c)
	if len(got) != 1 || got[0].Type == nil || got[0].Type.Name != "Foo" {
		t.Errorf("got %+v, want one struct Foo", got)
	}
}

func TestFilter_Kind_Method(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Sigs: []analyzer.Signature{
			{Name: "F", Kind: "func", Line: 1},
			{Name: "M", Kind: "method", Receiver: "(t *T)", Line: 2},
		},
	}})
	got := Filter{Kind: "method"}.Apply(c)
	if len(got) != 1 || got[0].Sig == nil || got[0].Sig.Name != "M" {
		t.Errorf("got %+v, want one method M", got)
	}
}

func TestFilter_Returns(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Sigs: []analyzer.Signature{
			{Name: "A", Kind: "func", Line: 1, ReturnTypes: []string{"error"}},
			{Name: "B", Kind: "func", Line: 2, ReturnTypes: []string{"int", "error"}},
			{Name: "C", Kind: "func", Line: 3, ReturnTypes: []string{"err error"}},
			{Name: "D", Kind: "func", Line: 4, ReturnTypes: []string{"int"}},
			{Name: "E", Kind: "func", Line: 5, ReturnTypes: nil},
		},
	}})
	got := Filter{Returns: "error"}.Apply(c)
	names := sigNames(got)
	want := []string{"A", "B", "C"}
	if !equalStringSets(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

func TestFilter_Takes(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Sigs: []analyzer.Signature{
			{Name: "A", Kind: "func", Line: 1, Parameters: []string{"ctx context.Context"}},
			{Name: "B", Kind: "func", Line: 2, Parameters: []string{"context.Context"}},
			{Name: "C", Kind: "func", Line: 3, Parameters: []string{"ctx context.Context", "id string"}},
			{Name: "D", Kind: "func", Line: 4, Parameters: []string{"id string"}},
		},
	}})
	got := Filter{Takes: "context.Context"}.Apply(c)
	names := sigNames(got)
	want := []string{"A", "B", "C"}
	if !equalStringSets(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, x := range a {
		m[x]++
	}
	for _, y := range b {
		m[y]--
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}

func TestFilter_Implements(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Sigs: []analyzer.Signature{
			// OrderService has all of Save+Load+Delete -> implements Repository
			{Name: "Save", Kind: "method", Receiver: "(o *OrderService)", Line: 10},
			{Name: "Load", Kind: "method", Receiver: "(o *OrderService)", Line: 11},
			{Name: "Delete", Kind: "method", Receiver: "(o *OrderService)", Line: 12},
			// ProductService has only Save+Load -> does NOT implement
			{Name: "Save", Kind: "method", Receiver: "(p *ProductService)", Line: 20},
			{Name: "Load", Kind: "method", Receiver: "(p *ProductService)", Line: 21},
		},
		Types: []analyzer.TypeDecl{
			{Name: "OrderService", Kind: "struct", Line: 5},
			{Name: "ProductService", Kind: "struct", Line: 15},
			{Name: "Repository", Kind: "interface", Line: 50,
				Methods: []analyzer.Method{
					{Name: "Save"}, {Name: "Load"}, {Name: "Delete"},
				}},
		},
	}})
	got := Filter{Implements: "Repository"}.Apply(c)
	names := sigNames(got)
	if len(names) != 1 || names[0] != "OrderService" {
		t.Errorf("got %v, want [OrderService]", names)
	}
}

func TestFilter_Implements_UnknownInterfaceReturnsEmpty(t *testing.T) {
	c := NewCorpus(nil)
	got := Filter{Implements: "Nope"}.Apply(c)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestFilter_HasReceiver(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Sigs: []analyzer.Signature{
			{Name: "Save", Kind: "method", Receiver: "(s *OrderService)", Line: 1},
			{Name: "Load", Kind: "method", Receiver: "(s OrderService)", Line: 2},
			{Name: "Delete", Kind: "method", Receiver: "(*OrderService)", Line: 3},
			{Name: "Other", Kind: "method", Receiver: "(p *ProductService)", Line: 4},
			{Name: "Free", Kind: "func", Line: 5},
		},
	}})
	got := Filter{HasReceiver: "OrderService"}.Apply(c)
	names := sigNames(got)
	if len(names) != 3 {
		t.Fatalf("got %v, want 3 OrderService methods", names)
	}
	wantSet := map[string]bool{"Save": true, "Load": true, "Delete": true}
	for _, n := range names {
		if !wantSet[n] {
			t.Errorf("unexpected method %q", n)
		}
	}
}

func TestFilter_HasMethod(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Sigs: []analyzer.Signature{
			{Name: "Delete", Kind: "method", Receiver: "(s *OrderService)", Line: 10},
			{Name: "Save", Kind: "method", Receiver: "(s *OrderService)", Line: 20},
			{Name: "Delete", Kind: "method", Receiver: "(p *ProductService)", Line: 30},
		},
		Types: []analyzer.TypeDecl{
			{Name: "OrderService", Kind: "struct", Line: 5},
			{Name: "ProductService", Kind: "struct", Line: 25},
			{Name: "UserService", Kind: "struct", Line: 50},
		},
	}})
	got := Filter{HasMethod: "Delete"}.Apply(c)
	names := sigNames(got)
	if len(names) != 2 {
		t.Fatalf("got %d matches, want 2: %v", len(names), names)
	}
	want := map[string]bool{"OrderService": true, "ProductService": true}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected match %q", n)
		}
	}
	// Each match's Methods must include the matched Delete.
	for _, m := range got {
		if len(m.Methods) != 1 || m.Methods[0].Name != "Delete" {
			t.Errorf("type %s: Methods = %+v, want one Delete", m.Type.Name, m.Methods)
		}
	}
}
