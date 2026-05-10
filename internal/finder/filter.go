package finder

import (
	"strings"

	"github.com/kypkk/acode/internal/analyzer"
)

// Filter holds the values of every find flag. Empty fields mean
// "no constraint on this dimension". All non-empty fields are AND'd.
type Filter struct {
	Kind        string
	HasMethod   string
	HasReceiver string
	Returns     string
	Takes       string
	Implements  string
}

// IsEmpty reports whether the filter has no constraints.
func (f Filter) IsEmpty() bool {
	return f.Kind == "" && f.HasMethod == "" && f.HasReceiver == "" &&
		f.Returns == "" && f.Takes == "" && f.Implements == ""
}

// Match is one row in the result set.
type Match struct {
	Path    string
	Package string

	// Type is set when the subject is a type declaration (struct / interface /
	// alias / named). Sig is set when the subject is a function or method.
	// Exactly one of these is non-nil per Match.
	Type *analyzer.TypeDecl
	Sig  *analyzer.Signature

	// Methods carries supporting child entries for type-grouped output
	// (e.g., the method that matched --has-method, or the methods returning
	// the requested type when --has-method is combined with --returns).
	Methods []analyzer.Signature
}

// Apply runs every set predicate on the corpus, AND-combined, and returns
// matching entries grouped by InferSubject(f).
func (f Filter) Apply(c *Corpus) []Match {
	subject := InferSubject(f)
	var out []Match
	for _, file := range c.Files {
		if subject == SubjectType {
			for i := range file.Types {
				t := &file.Types[i]
				if !f.matchType(t, c) {
					continue
				}
				m := Match{Path: file.Path, Package: file.Package, Type: t}
				m.Methods = f.relatedMethods(t, c)
				out = append(out, m)
			}
		} else {
			for i := range file.Sigs {
				s := &file.Sigs[i]
				if !f.matchSig(s) {
					continue
				}
				out = append(out, Match{Path: file.Path, Package: file.Package, Sig: s})
			}
		}
	}
	return out
}

func (f Filter) matchType(t *analyzer.TypeDecl, c *Corpus) bool {
	if f.Kind != "" && t.Kind != f.Kind {
		return false
	}
	if f.HasMethod != "" {
		if !typeHasMethod(c, t.Name, f.HasMethod) {
			return false
		}
	}
	if f.Implements != "" {
		info, ok := c.interfacesByName[f.Implements]
		if !ok {
			return false
		}
		if !methodSetSuperset(c.methodsByType[t.Name], info.MethodNames) {
			return false
		}
	}
	return true
}

// methodSetSuperset reports whether the methods on a type cover every method
// name in want. Name-only check, per spec.
func methodSetSuperset(have []analyzer.Signature, want []string) bool {
	got := map[string]bool{}
	for _, m := range have {
		got[m.Name] = true
	}
	for _, w := range want {
		if !got[w] {
			return false
		}
	}
	return true
}

func typeHasMethod(c *Corpus, typeName, methodName string) bool {
	for _, m := range c.methodsByType[typeName] {
		if m.Name == methodName {
			return true
		}
	}
	return false
}

// relatedMethods picks the methods of t that should display under it for
// type-grouped output. For now: only methods matching --has-method by name.
func (f Filter) relatedMethods(t *analyzer.TypeDecl, c *Corpus) []analyzer.Signature {
	if f.HasMethod == "" {
		return nil
	}
	var out []analyzer.Signature
	for _, m := range c.methodsByType[t.Name] {
		if m.Name == f.HasMethod {
			out = append(out, m)
		}
	}
	return out
}

func (f Filter) matchSig(s *analyzer.Signature) bool {
	if f.Kind != "" && s.Kind != f.Kind {
		return false
	}
	if f.HasReceiver != "" {
		if s.Kind != "method" {
			return false
		}
		if extractReceiverTypeName(s.Receiver) != f.HasReceiver {
			return false
		}
	}
	if f.Returns != "" {
		if !anyContains(s.ReturnTypes, f.Returns) {
			return false
		}
	}
	if f.Takes != "" {
		if !anyContains(s.Parameters, f.Takes) {
			return false
		}
	}
	return true
}

func anyContains(items []string, needle string) bool {
	for _, s := range items {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
