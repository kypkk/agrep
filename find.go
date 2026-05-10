package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kypkk/agrep/internal/finder"
)

// FindOptions mirrors the cobra flags so runFind can be unit-tested without
// driving the cobra layer.
type FindOptions struct {
	In          string
	Kind        string
	HasMethod   string
	HasReceiver string
	Returns     string
	Takes       string
	Implements  string
	Limit       int
	All         bool
}

var findOpts FindOptions

var findCmd = &cobra.Command{
	Use:   "find",
	Short: "Cross-file structural search",
	Long: `find searches across .go files for declarations matching structural
predicates that grep cannot express — interface implementation, method
ownership, return-type containment, etc.

Combine flags with AND semantics; at least one constraint is required.

Examples:
  agrep find --has-method=Delete --in=internal/service/
  agrep find --returns=error --in=internal/util/
  agrep find --implements=Repository --in=internal/
  agrep find --kind=interface --in=internal/repository/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFind(findOpts, cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

func init() {
	findCmd.Flags().StringVar(&findOpts.In, "in", ".", "Directory or .go file to search")
	findCmd.Flags().StringVar(&findOpts.Kind, "kind", "", "Filter: declaration kind (func, method, struct, interface, alias, named)")
	findCmd.Flags().StringVar(&findOpts.HasMethod, "has-method", "", "Filter: types with any method name containing X (substring, case-insensitive)")
	findCmd.Flags().StringVar(&findOpts.HasReceiver, "has-receiver", "", "Filter: methods whose receiver type-name contains X (substring, case-insensitive)")
	findCmd.Flags().StringVar(&findOpts.Returns, "returns", "", "Filter: substring match against return-type slice (case-insensitive)")
	findCmd.Flags().StringVar(&findOpts.Takes, "takes", "", "Filter: substring match against parameter slice (case-insensitive)")
	findCmd.Flags().StringVar(&findOpts.Implements, "implements", "", "Filter: types whose method-name set is a superset of interface X's")
	findCmd.Flags().IntVar(&findOpts.Limit, "limit", 20, "Max results to display")
	findCmd.Flags().BoolVar(&findOpts.All, "all", false, "Disable pagination limit")
	rootCmd.AddCommand(findCmd)
}

var validKinds = map[string]bool{
	"func": true, "method": true, "struct": true, "interface": true,
	"alias": true, "named": true,
}

func runFind(opts FindOptions, stdout, stderr io.Writer) error {
	filter := finder.Filter{
		Kind:        opts.Kind,
		HasMethod:   opts.HasMethod,
		HasReceiver: opts.HasReceiver,
		Returns:     opts.Returns,
		Takes:       opts.Takes,
		Implements:  opts.Implements,
	}
	if filter.IsEmpty() {
		return fmt.Errorf("no filter criteria; pass at least one of --kind / --has-method / --has-receiver / --returns / --takes / --implements")
	}
	if opts.Kind != "" && !validKinds[opts.Kind] {
		return fmt.Errorf("invalid --kind %q (want func, method, struct, interface, alias, named)", opts.Kind)
	}

	in := opts.In
	if in == "" {
		in = "."
	}
	paths, err := finder.Walk(in)
	if err != nil {
		return err
	}
	files := make([]finder.FileResult, 0, len(paths))
	for _, p := range paths {
		fr, err := finder.ScanFile(p)
		if err != nil {
			fmt.Fprintf(stderr, "agrep: skipping %s: %v\n", p, err)
			continue
		}
		files = append(files, fr)
	}
	corpus := finder.NewCorpus(files)
	matches := filter.Apply(corpus)

	if len(matches) == 0 {
		renderNoResults(stdout, opts, filter, corpus, len(paths))
		return nil
	}

	subject := finder.InferSubject(filter)
	limit := opts.Limit
	if opts.All || limit <= 0 {
		limit = len(matches)
	}
	renderMatches(stdout, matches, subject, limit, filter)
	return nil
}

func renderMatches(w io.Writer, matches []finder.Match, subject finder.Subject, limit int, f finder.Filter) {
	header := matchHeader(matches, subject, f)
	fmt.Fprintln(w, header)
	fmt.Fprintln(w)

	shown := matches
	if len(shown) > limit {
		shown = shown[:limit]
	}
	if subject == finder.SubjectType {
		renderTypeGroups(w, shown)
	} else {
		renderFlat(w, shown)
	}
	fmt.Fprintln(w)
	if len(matches) > limit {
		fmt.Fprintf(w, "%d results (use --all to see all)\n", len(matches))
	} else {
		fmt.Fprintf(w, "%d results\n", len(matches))
	}
}

func matchHeader(matches []finder.Match, subject finder.Subject, f finder.Filter) string {
	n := len(matches)
	switch {
	case f.HasMethod != "":
		return fmt.Sprintf("%d types with method %s:", n, f.HasMethod)
	case f.Implements != "":
		return fmt.Sprintf("%d types implementing %s:", n, f.Implements)
	case f.Returns != "":
		return fmt.Sprintf("%d functions return %s:", n, f.Returns)
	case f.Takes != "":
		return fmt.Sprintf("%d functions take %s:", n, f.Takes)
	case f.HasReceiver != "":
		return fmt.Sprintf("%d methods on %s:", n, f.HasReceiver)
	case subject == finder.SubjectType:
		return fmt.Sprintf("%d types of kind %s:", n, f.Kind)
	default:
		return fmt.Sprintf("%d matches:", n)
	}
}

func renderTypeGroups(w io.Writer, matches []finder.Match) {
	for _, m := range matches {
		fmt.Fprintf(w, "%-22s %s:%d\n", m.Type.Name, m.Path, m.Type.Line)
		for _, child := range m.Methods {
			fmt.Fprintf(w, "  func %s %s(%s)%s :%d\n",
				child.Receiver, child.Name,
				strings.Join(child.Parameters, ", "),
				renderReturns(child.ReturnTypes),
				child.Line)
		}
		fmt.Fprintln(w)
	}
}

func renderFlat(w io.Writer, matches []finder.Match) {
	// Sort flat matches by path then line for stable output.
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Path != matches[j].Path {
			return matches[i].Path < matches[j].Path
		}
		return entityLine(matches[i]) < entityLine(matches[j])
	})
	for _, m := range matches {
		if m.Sig != nil {
			label := "func"
			if m.Sig.Kind == "method" {
				label = "method " + m.Sig.Receiver
			}
			fmt.Fprintf(w, "%s:%d   %s %s(%s)%s\n",
				m.Path, m.Sig.Line, label, m.Sig.Name,
				strings.Join(m.Sig.Parameters, ", "),
				renderReturns(m.Sig.ReturnTypes))
		} else if m.Type != nil {
			fmt.Fprintf(w, "%s:%d   %s %s\n", m.Path, m.Type.Line, m.Type.Kind, m.Type.Name)
		}
	}
}

func entityLine(m finder.Match) int {
	if m.Sig != nil {
		return m.Sig.Line
	}
	if m.Type != nil {
		return m.Type.Line
	}
	return 0
}

func renderReturns(rs []string) string {
	if len(rs) == 0 {
		return ""
	}
	if len(rs) == 1 {
		return " " + rs[0]
	}
	return " (" + strings.Join(rs, ", ") + ")"
}

func renderNoResults(w io.Writer, opts FindOptions, f finder.Filter, c *finder.Corpus, nFiles int) {
	fmt.Fprintln(w, "No results.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Searched %d files in %s for %s.\n", nFiles, opts.In, queryDescription(f))
	suggestions := finder.Suggest(f, c)
	if len(suggestions) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Did you mean:")
	for _, s := range suggestions {
		fmt.Fprintf(w, "  - %s=%s\n", s.Field, s.Value)
	}
}

func queryDescription(f finder.Filter) string {
	switch {
	case f.HasMethod != "":
		return fmt.Sprintf("types with method %q", f.HasMethod)
	case f.Implements != "":
		return fmt.Sprintf("types implementing %q", f.Implements)
	case f.Returns != "":
		return fmt.Sprintf("functions returning %q", f.Returns)
	case f.Takes != "":
		return fmt.Sprintf("functions taking %q", f.Takes)
	case f.HasReceiver != "":
		return fmt.Sprintf("methods on %q", f.HasReceiver)
	case f.Kind != "":
		return fmt.Sprintf("declarations of kind %s", f.Kind)
	}
	return "matching declarations"
}
