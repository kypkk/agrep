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

Two output formats are supported:
  human (default)  Indented, optionally coloured for terminal viewing.
  agent            Dense, deterministic, one entity per line — suitable
                   for piping into other tools or another agent.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSignatures(args[0], sigFormat, sigAll, cmd.OutOrStdout())
	},
}

func init() {
	signaturesCmd.Flags().StringVar(&sigFormat, "format", "human", "Output format: human or agent")
	signaturesCmd.Flags().BoolVar(&sigAll, "all", false, "Include unexported (lowercase-prefix) symbols")
	rootCmd.AddCommand(signaturesCmd)
}

// runSignatures is the testable core of the signatures command. It is kept
// separate from cobra wiring so tests can exercise it directly without
// touching the global command tree or stdout.
func runSignatures(path, formatName string, all bool, out io.Writer) error {
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
	if !all {
		sigs = filterExportedSignatures(sigs)
		types = filterExportedTypes(types)
	}

	var rendered string
	switch formatName {
	case "human":
		rendered = format.Human(sigs, types, format.HumanOptions{Color: writerIsTTY(out)})
	case "agent":
		rendered = format.Agent(sigs, types)
	default:
		return fmt.Errorf("unknown --format %q (want human or agent)", formatName)
	}

	if _, err := io.WriteString(out, rendered); err != nil {
		return err
	}
	// Human format omits a trailing newline so multiple invocations don't
	// stack blank lines. Add one here so terminal prompts land on a fresh
	// line. Agent already ends each entity in '\n' so its output is clean.
	if rendered != "" && rendered[len(rendered)-1] != '\n' {
		if _, err := io.WriteString(out, "\n"); err != nil {
			return err
		}
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
