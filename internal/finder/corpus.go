package finder

import (
	"strings"

	"github.com/kypkk/agrep/internal/analyzer"
)

// Corpus aggregates FileResults and pre-computes the indexes the filter and
// suggester need.
type Corpus struct {
	Files []FileResult

	// methodsByType maps a type's name (extracted from the receiver) to the
	// signatures of all methods on it, possibly across multiple files.
	methodsByType map[string][]analyzer.Signature

	// interfacesByName maps interface name to its declaration site and method
	// name list. Used by --implements.
	interfacesByName map[string]InterfaceInfo

	// Index of distinct names seen, for the suggester.
	allMethodNames    []string
	allTypeNames      []string
	allInterfaceNames []string
	allReceiverNames  []string
}

// InterfaceInfo identifies an interface declaration site and its method names.
type InterfaceInfo struct {
	Path        string
	Line        int
	MethodNames []string
}

// NewCorpus aggregates a set of FileResults and computes derived indexes.
func NewCorpus(files []FileResult) *Corpus {
	c := &Corpus{
		Files:            files,
		methodsByType:    map[string][]analyzer.Signature{},
		interfacesByName: map[string]InterfaceInfo{},
	}
	methodNames := map[string]struct{}{}
	typeNames := map[string]struct{}{}
	interfaceNames := map[string]struct{}{}
	receiverNames := map[string]struct{}{}

	for _, f := range files {
		for _, s := range f.Sigs {
			if s.Kind == "method" {
				if t := extractReceiverTypeName(s.Receiver); t != "" {
					c.methodsByType[t] = append(c.methodsByType[t], s)
					receiverNames[t] = struct{}{}
				}
			}
			methodNames[s.Name] = struct{}{}
		}
		for _, t := range f.Types {
			typeNames[t.Name] = struct{}{}
			if t.Kind == "interface" {
				interfaceNames[t.Name] = struct{}{}
				names := make([]string, 0, len(t.Methods))
				for _, m := range t.Methods {
					names = append(names, m.Name)
				}
				c.interfacesByName[t.Name] = InterfaceInfo{
					Path:        f.Path,
					Line:        t.Line,
					MethodNames: names,
				}
			}
		}
	}
	c.allMethodNames = setKeys(methodNames)
	c.allTypeNames = setKeys(typeNames)
	c.allInterfaceNames = setKeys(interfaceNames)
	c.allReceiverNames = setKeys(receiverNames)
	return c
}

// extractReceiverTypeName turns "(s *OrderService)" / "(*Box[T])" / "(o T)"
// into "OrderService" / "Box" / "T". Returns "" for empty or malformed input.
func extractReceiverTypeName(receiver string) string {
	s := strings.TrimSpace(receiver)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Split on whitespace to drop the optional variable name.
	if i := strings.LastIndex(s, " "); i >= 0 {
		s = s[i+1:]
	}
	s = strings.TrimPrefix(s, "*")
	// Drop generic type parameters: Box[T] -> Box.
	if i := strings.Index(s, "["); i >= 0 {
		s = s[:i]
	}
	return s
}

func setKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
