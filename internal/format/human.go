package format

import (
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"

	"github.com/kypkk/agrep/internal/analyzer"
)

// HumanOptions configures the human-facing renderer.
type HumanOptions struct {
	// Color enables ANSI colour output. Off by default so test runs and piped
	// output stay deterministic without relying on TTY detection.
	Color bool
}

// Human renders signatures and type declarations for terminal viewing:
// doc comments above each entity, indented bodies, and one blank line between
// entities. The output is sorted by (Line, Name) so it tracks source order.
//
// When file is non-empty, the output is prefixed with two header lines —
// `file: <path>` and `package: <name>` — separated from the body by a blank
// line. file == "" skips the header.
func Human(file, pkg string, sigs []analyzer.Signature, types []analyzer.TypeDecl, opts HumanOptions) string {
	p := newPalette(opts.Color)
	type entry struct {
		line int
		name string
		text string
	}
	entries := make([]entry, 0, len(sigs)+len(types))
	for _, s := range sigs {
		entries = append(entries, entry{line: s.Line, name: s.Name, text: p.humanFunc(s)})
	}
	for _, t := range types {
		entries = append(entries, entry{line: t.Line, name: t.Name, text: p.humanType(t)})
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
		if len(entries) > 0 {
			b.WriteString("\n\n")
		}
	}
	for i, e := range entries {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(e.text)
	}
	return b.String()
}

// palette holds colorize closures for each visual role. When Color is false
// every closure is the identity function, so the output contains no ANSI
// escapes regardless of TTY state.
type palette struct {
	typeName func(string) string
	funcName func(string) string
	kind     func(string) string
	line     func(string) string
	doc      func(string) string
}

func newPalette(useColor bool) *palette {
	mk := func(attrs ...color.Attribute) func(string) string {
		if !useColor {
			return func(s string) string { return s }
		}
		c := color.New(attrs...)
		c.EnableColor()
		return func(s string) string { return c.Sprint(s) }
	}
	return &palette{
		typeName: mk(color.FgMagenta, color.Bold),
		funcName: mk(color.FgCyan, color.Bold),
		kind:     mk(color.Faint),
		line:     mk(color.Faint),
		doc:      mk(color.FgGreen),
	}
}

func (p *palette) humanFunc(s analyzer.Signature) string {
	var b strings.Builder
	for _, line := range docLines(s.DocComment) {
		b.WriteString(p.doc("// " + line))
		b.WriteByte('\n')
	}
	// `method (recv) Name(...)` for methods, `func Name(...)` for plain
	// functions. Mirrors the agent format keyword choice so consumers
	// scanning either output use the same vocabulary. Empty Kind falls
	// through to "func" — matches the JSON formatter's defensive default.
	if s.Kind == "method" {
		b.WriteString(p.kind("method "))
		if s.Receiver != "" {
			b.WriteString(s.Receiver)
			b.WriteByte(' ')
		}
	} else {
		b.WriteString(p.kind("func "))
	}
	b.WriteString(p.funcName(s.Name))
	b.WriteByte('(')
	b.WriteString(strings.Join(s.Parameters, ", "))
	b.WriteByte(')')
	if rs := humanReturns(s.ReturnTypes); rs != "" {
		b.WriteByte(' ')
		b.WriteString(rs)
	}
	b.WriteString("  ")
	b.WriteString(p.line(humanLine(s.Line)))
	return b.String()
}

func humanReturns(rs []string) string {
	switch len(rs) {
	case 0:
		return ""
	case 1:
		return rs[0]
	default:
		return "(" + strings.Join(rs, ", ") + ")"
	}
}

func humanLine(n int) string {
	return "# line " + strconv.Itoa(n)
}

func (p *palette) humanType(t analyzer.TypeDecl) string {
	switch t.Kind {
	case "struct":
		return p.humanStruct(t)
	case "interface":
		return p.humanInterface(t)
	case "alias":
		return p.humanAlias(t)
	case "named":
		return p.humanNamed(t)
	}
	return p.kind(t.Kind+" ") + p.typeName(t.Name) + "  " + p.line(humanLine(t.Line))
}

func (p *palette) humanStruct(t analyzer.TypeDecl) string {
	var b strings.Builder
	b.WriteString(p.kind("struct "))
	b.WriteString(p.typeName(t.Name))
	b.WriteString("  ")
	b.WriteString(p.line(humanLine(t.Line)))
	if len(t.Fields) == 0 {
		return b.String()
	}
	width := 0
	for _, f := range t.Fields {
		if len(f.Name) > width {
			width = len(f.Name)
		}
	}
	for _, f := range t.Fields {
		b.WriteString("\n  ")
		if f.Name == "" {
			b.WriteString(f.Type)
			continue
		}
		b.WriteString(f.Name)
		b.WriteString(strings.Repeat(" ", width-len(f.Name)+1))
		b.WriteString(f.Type)
	}
	return b.String()
}

func (p *palette) humanInterface(t analyzer.TypeDecl) string {
	var b strings.Builder
	b.WriteString(p.kind("interface "))
	b.WriteString(p.typeName(t.Name))
	b.WriteString("  ")
	b.WriteString(p.line(humanLine(t.Line)))
	for _, m := range t.Methods {
		b.WriteString("\n  ")
		b.WriteString(m.Name)
		b.WriteByte('(')
		b.WriteString(strings.Join(m.Parameters, ", "))
		b.WriteByte(')')
		if rs := humanReturns(m.ReturnTypes); rs != "" {
			b.WriteByte(' ')
			b.WriteString(rs)
		}
	}
	return b.String()
}

func (p *palette) humanAlias(t analyzer.TypeDecl) string {
	return p.kind("type ") + p.typeName(t.Name) + " = " + t.Underlying + "  " + p.line(humanLine(t.Line))
}

func (p *palette) humanNamed(t analyzer.TypeDecl) string {
	return p.kind("type ") + p.typeName(t.Name) + " " + t.Underlying + "  " + p.line(humanLine(t.Line))
}

func docLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
