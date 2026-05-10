package finder

import (
	"strings"
	"testing"

	"github.com/kypkk/acode/internal/analyzer"
)

func TestSuggest_HasMethod_LevenshteinAndSubstring(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Sigs: []analyzer.Signature{
			{Name: "Create", Kind: "method", Receiver: "(t *T)"},
			{Name: "Update", Kind: "method", Receiver: "(t *T)"},
			{Name: "Delete", Kind: "method", Receiver: "(t *T)"},
		},
	}})
	// "Cretae" is one transposition off Create -> Levenshtein 2.
	got := Suggest(Filter{HasMethod: "Cretae"}, c)
	if len(got) == 0 {
		t.Fatalf("expected at least one suggestion, got 0")
	}
	found := false
	for _, s := range got {
		if s.Field == "method" && s.Value == "Create" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected suggestion method=Create, got %+v", got)
	}
}

func TestSuggest_Implements_FromInterfaceNames(t *testing.T) {
	c := NewCorpus([]FileResult{{
		Path: "a.go",
		Types: []analyzer.TypeDecl{
			{Name: "Repository", Kind: "interface", Methods: []analyzer.Method{{Name: "Get"}}},
			{Name: "Reader", Kind: "interface", Methods: []analyzer.Method{{Name: "Read"}}},
		},
	}})
	got := Suggest(Filter{Implements: "Repo"}, c)
	hint := ""
	for _, s := range got {
		if s.Field == "implements" {
			hint = s.Value
		}
	}
	if !strings.Contains(hint, "Repository") {
		t.Errorf("expected Repository in suggestion, got %+v", got)
	}
}

func TestSuggest_CapsAtThreePerField(t *testing.T) {
	names := []string{"Aaaa", "Aaab", "Aaac", "Aaad", "Aaae"}
	sigs := make([]analyzer.Signature, len(names))
	for i, n := range names {
		sigs[i] = analyzer.Signature{Name: n, Kind: "method", Receiver: "(t *T)"}
	}
	c := NewCorpus([]FileResult{{Path: "a.go", Sigs: sigs}})
	got := Suggest(Filter{HasMethod: "Aaa"}, c)
	count := 0
	for _, s := range got {
		if s.Field == "method" {
			count++
		}
	}
	if count > 3 {
		t.Errorf("got %d method suggestions, want ≤ 3", count)
	}
}
