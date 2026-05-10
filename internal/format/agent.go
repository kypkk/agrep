// Package format renders analyzer output for either humans or agents.
//
// Agent output is dense, deterministic, and free of ANSI escapes — the
// contract is that identical input always produces byte-identical output, so
// downstream agents and caches can compare safely.
//
// Human output uses indentation, optional color, and a small amount of
// decorative spacing for terminal viewing.
package format

import (
	"sort"
	"strconv"
	"strings"

	"github.com/kypkk/acode/internal/analyzer"
)

// Agent renders signatures and type declarations in a dense one-line-per-entity
// form, sorted by (Line, Name). Each line ends in a single LF; the result has
// no other whitespace and no ANSI codes.
//
// When file is non-empty, the output is prefixed with two header lines —
// `file: <path>` and `package: <name>` — so downstream consumers always know
// which file produced the rest. file == "" skips the header (used by tests
// that exercise rendering in isolation).
func Agent(file, pkg string, sigs []analyzer.Signature, types []analyzer.TypeDecl) string {
	type entry struct {
		line int
		name string
		text string
	}
	entries := make([]entry, 0, len(sigs)+len(types))
	for _, s := range sigs {
		entries = append(entries, entry{line: s.Line, name: s.Name, text: agentFunc(s)})
	}
	for _, t := range types {
		entries = append(entries, entry{line: t.Line, name: t.Name, text: agentType(t)})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].line != entries[j].line {
			return entries[i].line < entries[j].line
		}
		return entries[i].name < entries[j].name
	})
	var b strings.Builder
	if file != "" {
		b.WriteString("file: ")
		b.WriteString(file)
		b.WriteByte('\n')
		b.WriteString("package: ")
		b.WriteString(pkg)
		b.WriteByte('\n')
	}
	for _, e := range entries {
		b.WriteString(e.text)
		b.WriteByte('\n')
	}
	return b.String()
}

func agentFunc(s analyzer.Signature) string {
	if s.Kind == "method" {
		return "method " + strconv.Itoa(s.Line) + " " + s.Receiver + " " + s.Name +
			"(" + strings.Join(s.Parameters, ", ") + ")" +
			agentReturns(s.ReturnTypes)
	}
	return "func " + strconv.Itoa(s.Line) + " " + s.Name +
		"(" + strings.Join(s.Parameters, ", ") + ")" +
		agentReturns(s.ReturnTypes)
}

// agentReturns renders 0/1/N returns with the same shape Go uses in source:
// nothing, a bare type, or a parenthesised list.
func agentReturns(rs []string) string {
	switch len(rs) {
	case 0:
		return ""
	case 1:
		return " " + rs[0]
	default:
		return " (" + strings.Join(rs, ", ") + ")"
	}
}

func agentType(t analyzer.TypeDecl) string {
	head := t.Kind + " " + strconv.Itoa(t.Line) + " " + t.Name
	switch t.Kind {
	case "struct":
		return head + " {" + agentFields(t.Fields) + "}"
	case "interface":
		return head + " {" + agentMethods(t.Methods) + "}"
	case "alias":
		return head + " = " + t.Underlying
	case "named":
		return head + " " + t.Underlying
	}
	return head
}

func agentFields(fs []analyzer.Field) string {
	parts := make([]string, len(fs))
	for i, f := range fs {
		if f.Name == "" {
			parts[i] = f.Type
		} else {
			parts[i] = f.Name + " " + f.Type
		}
	}
	return strings.Join(parts, "; ")
}

func agentMethods(ms []analyzer.Method) string {
	parts := make([]string, len(ms))
	for i, m := range ms {
		parts[i] = m.Name + "(" + strings.Join(m.Parameters, ", ") + ")" + agentReturns(m.ReturnTypes)
	}
	return strings.Join(parts, "; ")
}
