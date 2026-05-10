package finder

import (
	"reflect"
	"sort"
	"testing"

	"github.com/kypkk/acode/internal/analyzer"
)

func TestExtractReceiverTypeName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"(s *OrderService)", "OrderService"},
		{"(o OrderService)", "OrderService"},
		{"(*OrderService)", "OrderService"},
		{"(OrderService)", "OrderService"},
		{"(b *Box[T])", "Box"},
		{"(b Box[T])", "Box"},
		{"", ""},
		{"()", ""},
	}
	for _, c := range cases {
		if got := extractReceiverTypeName(c.in); got != c.want {
			t.Errorf("extractReceiverTypeName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNewCorpus_BuildsIndexes(t *testing.T) {
	files := []FileResult{
		{
			Path: "a.go", Package: "x",
			Sigs: []analyzer.Signature{
				{Name: "Save", Kind: "method", Receiver: "(o *OrderService)", Line: 10},
				{Name: "Load", Kind: "method", Receiver: "(o *OrderService)", Line: 20},
				{Name: "Free", Kind: "func", Line: 30},
			},
			Types: []analyzer.TypeDecl{
				{Name: "OrderService", Kind: "struct", Line: 5},
				{Name: "Repository", Kind: "interface", Line: 50,
					Methods: []analyzer.Method{{Name: "Save"}, {Name: "Load"}}},
			},
		},
		{
			Path: "b.go", Package: "x",
			Sigs: []analyzer.Signature{
				{Name: "Delete", Kind: "method", Receiver: "(o *OrderService)", Line: 5},
			},
		},
	}
	c := NewCorpus(files)

	// methodsByType: OrderService should have 3 methods across the two files.
	got := c.methodsByType["OrderService"]
	if len(got) != 3 {
		t.Errorf("methodsByType[OrderService] len = %d, want 3 (got %+v)", len(got), got)
	}

	// interfacesByName: Repository
	if info, ok := c.interfacesByName["Repository"]; !ok {
		t.Errorf("interfacesByName missing Repository")
	} else {
		want := []string{"Save", "Load"}
		sort.Strings(info.MethodNames)
		sort.Strings(want)
		if !reflect.DeepEqual(info.MethodNames, want) {
			t.Errorf("Repository methods = %v, want %v", info.MethodNames, want)
		}
	}

	// allMethodNames includes Save, Load, Free, Delete
	got2 := append([]string{}, c.allMethodNames...)
	sort.Strings(got2)
	want2 := []string{"Delete", "Free", "Load", "Save"}
	if !reflect.DeepEqual(got2, want2) {
		t.Errorf("allMethodNames = %v, want %v", got2, want2)
	}

	// allInterfaceNames includes Repository
	if !contains(c.allInterfaceNames, "Repository") {
		t.Errorf("allInterfaceNames missing Repository: %v", c.allInterfaceNames)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
