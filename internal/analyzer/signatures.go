// Package analyzer extracts structured information from parse trees.
package analyzer

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/kypkk/acode/internal/parser"
)

// Signature describes a top-level function or method declaration.
//
// v0 only populates Name and Line. The other fields are reserved for
// subsequent tickets and are documented here so the shape of the struct
// stays stable as features land.
type Signature struct {
	Name        string
	Kind        string
	Receiver    string
	Parameters  []string
	ReturnTypes []string
	DocComment  string
	IsExported  bool
	Line        int
}

// ExtractSignatures walks the top level of a Go parse tree and returns one
// Signature per function_declaration and method_declaration. Anonymous
// function literals are intentionally not included — they are not top-level
// declarations.
func ExtractSignatures(tree *parser.Tree, src []byte) []Signature {
	var sigs []Signature
	root := tree.RootNode()
	count := int(root.NamedChildCount())
	for i := 0; i < count; i++ {
		child := root.NamedChild(i)
		switch child.Type() {
		case "function_declaration", "method_declaration":
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			kind := "func"
			receiver := ""
			if child.Type() == "method_declaration" {
				kind = "method"
				if rcv := child.ChildByFieldName("receiver"); rcv != nil {
					receiver = rcv.Content(src)
				}
			}
			sigs = append(sigs, Signature{
				Name:        nameNode.Content(src),
				Kind:        kind,
				Receiver:    receiver,
				Line:        int(child.StartPoint().Row) + 1,
				Parameters:  extractParameters(child.ChildByFieldName("parameters"), src),
				ReturnTypes: extractReturnTypes(child.ChildByFieldName("result"), src),
				DocComment:  extractDocComment(root, i, child, src),
			})
		}
	}
	return sigs
}

// extractParameters reads each parameter_declaration / variadic_parameter_declaration
// child of a parameter_list and returns its source text. Multi-name shared-type
// declarations like `x, y int` stay as a single entry by design — expanding them
// is a v1 concern.
func extractParameters(list *sitter.Node, src []byte) []string {
	if list == nil {
		return nil
	}
	var out []string
	for i := 0; i < int(list.NamedChildCount()); i++ {
		c := list.NamedChild(i)
		switch c.Type() {
		case "parameter_declaration", "variadic_parameter_declaration":
			out = append(out, c.Content(src))
		}
	}
	return out
}

// extractDocComment walks backward from declIdx through parent's named
// children, gathering consecutive `//` line comments that touch the
// declaration with no blank line between. The collected lines are joined
// with "\n" and have the `//` prefix (plus at most one space) stripped,
// matching the godoc convention. Block comments (`/* */`) are not picked
// up by design — the spec scopes this to line comments only.
func extractDocComment(parent *sitter.Node, declIdx int, decl *sitter.Node, src []byte) string {
	var lines []string
	expectedEndRow := int(decl.StartPoint().Row) - 1
	for j := declIdx - 1; j >= 0; j-- {
		sib := parent.NamedChild(j)
		if sib.Type() != "comment" {
			break
		}
		text := sib.Content(src)
		if !strings.HasPrefix(text, "//") {
			break
		}
		if int(sib.EndPoint().Row) != expectedEndRow {
			break
		}
		lines = append(lines, stripLineCommentPrefix(text))
		expectedEndRow = int(sib.StartPoint().Row) - 1
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return strings.Join(lines, "\n")
}

// stripLineCommentPrefix removes the `//` and at most one space after it,
// preserving any further indentation the author wrote intentionally.
func stripLineCommentPrefix(s string) string {
	s = strings.TrimPrefix(s, "//")
	if strings.HasPrefix(s, " ") {
		s = s[1:]
	}
	return s
}

// extractReturnTypes handles all three forms a function result can take:
// no result (nil), a bare type (`func() int`), or a parenthesised list which
// the grammar models as a parameter_list (`func() (int, error)` or named).
func extractReturnTypes(result *sitter.Node, src []byte) []string {
	if result == nil {
		return nil
	}
	if result.Type() == "parameter_list" {
		return extractParameters(result, src)
	}
	return []string{result.Content(src)}
}
