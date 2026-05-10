package analyzer

import "github.com/kypkk/agrep/internal/parser"

// PackageName returns the Go package name declared at the top of the file,
// or "" if no package_clause is present (malformed source).
func PackageName(tree *parser.Tree, src []byte) string {
	root := tree.RootNode()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() != "package_clause" {
			continue
		}
		for j := 0; j < int(child.NamedChildCount()); j++ {
			id := child.NamedChild(j)
			if id.Type() == "package_identifier" {
				return id.Content(src)
			}
		}
	}
	return ""
}
