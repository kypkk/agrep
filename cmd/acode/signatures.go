package main

import (
	"fmt"
	"io"
	"os"
	"unicode"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/kypkk/acode/internal/analyzer"
	"github.com/kypkk/acode/internal/format"
	"github.com/kypkk/acode/internal/parser"
)

var (
	sigFormat string
	sigAll    bool
)

var signaturesCmd = &cobra.Command{
	Use:   "signatures <file>",
	Short: "Extract signatures and types from a Go source file",
	Long: `signatures parses a Go source file and prints its top-level function
signatures, type declarations (struct, interface, alias, named), and doc
comments.

By default only exported (capital-prefixed) names are shown. Pass --all
to include unexported symbols. The members of an included type — struct
fields and interface methods — are always shown in full.

Three output formats are supported:
  human (default)  Indented, optionally coloured for terminal viewing.
  agent            Dense, deterministic, one entity per line — suitable
                   for piping into other tools or another agent.
  json             Structured JSON document for programmatic consumers
                   (MCP servers, future skills). Schema is committed to
                   be additive across versions.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSignatures(args[0], sigFormat, sigAll, cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

func init() {
	signaturesCmd.Flags().StringVar(&sigFormat, "format", "human", "Output format: human, agent, or json")
	signaturesCmd.Flags().BoolVar(&sigAll, "all", false, "Include unexported (lowercase-prefix) symbols")
	rootCmd.AddCommand(signaturesCmd)
}

// runSignatures is the testable core of the signatures command. It is kept
// separate from cobra wiring so tests can exercise it directly without
// touching the global command tree or stdout.
//
// stdout receives the formatted output. stderr receives diagnostic hints
// (e.g., "everything was filtered, try --all"). Splitting the streams keeps
// stdout clean for piping into jq or another agent.
func runSignatures(path, formatName string, all bool, stdout, stderr io.Writer) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	tree, err := parser.NewGoParser().Parse(src)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	sigs := analyzer.ExtractSignatures(tree, src)
	types := analyzer.ExtractTypes(tree, src)
	totalBefore := len(sigs) + len(types)
	if !all {
		sigs = filterExportedSignatures(sigs)
		types = filterExportedTypes(types)
	}
	totalAfter := len(sigs) + len(types)

	var rendered string
	switch formatName {
	case "human":
		rendered = format.Human(sigs, types, format.HumanOptions{Color: writerIsTTY(stdout)})
	case "agent":
		rendered = format.Agent(sigs, types)
	case "json":
		rendered = format.JSON(path, analyzer.PackageName(tree, src), sigs, types)
	default:
		return fmt.Errorf("unknown --format %q (want human, agent, or json)", formatName)
	}

	if _, err := io.WriteString(stdout, rendered); err != nil {
		return err
	}
	// Human format omits a trailing newline so multiple invocations don't
	// stack blank lines. Add one here so terminal prompts land on a fresh
	// line. Agent already ends each entity in '\n' so its output is clean.
	if rendered != "" && rendered[len(rendered)-1] != '\n' {
		if _, err := io.WriteString(stdout, "\n"); err != nil {
			return err
		}
	}

	// Hint: silent failure is bad UX. If the filter swallowed every entry,
	// tell the user on stderr so they don't think acode is broken. Common
	// trigger: package main with only lowercase decls (`func main`, helpers).
	if !all && totalAfter == 0 && totalBefore > 0 {
		fmt.Fprintf(stderr,
			"acode: no exported declarations in this file (%d unexported skipped). Re-run with --all to include them.\n",
			totalBefore)
	}
	return nil
}

func filterExportedSignatures(sigs []analyzer.Signature) []analyzer.Signature {
	out := sigs[:0:0]
	for _, s := range sigs {
		if isExportedName(s.Name) {
			out = append(out, s)
		}
	}
	return out
}

func filterExportedTypes(types []analyzer.TypeDecl) []analyzer.TypeDecl {
	out := types[:0:0]
	for _, t := range types {
		if isExportedName(t.Name) {
			out = append(out, t)
		}
	}
	return out
}

// isExportedName reports whether the first rune of name is uppercase, the
// Go convention for visibility outside its package.
func isExportedName(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

func writerIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}
