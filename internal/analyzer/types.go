package analyzer

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/kypkk/acode/internal/parser"
)

// TypeDecl describes a top-level Go type declaration. Kind discriminates the
// shape so consumers know which of Fields / Methods / Underlying to look at.
//
// v0 populates Name, Kind, Line, and exactly one of {Fields, Methods,
// Underlying}. DocComment / IsExported are reserved for a follow-up ticket.
type TypeDecl struct {
	Name       string
	Kind       string
	Line       int
	Fields     []Field
	Methods    []Method
	Underlying string
}

// Field is one entry in a struct. Multi-name declarations like
// `Name, Email string` expand into one Field per name. Embedded fields have
// Name == "" and Type set to the embedded type expression.
type Field struct {
	Name string
	Type string
}

// Method is one method signature inside an interface. Same shape as the
// per-call slices on Signature so downstream formatters can render either.
type Method struct {
	Name        string
	Parameters  []string
	ReturnTypes []string
}

// ExtractTypes walks the top level of a Go parse tree and returns one
// TypeDecl per type_spec or type_alias inside any type_declaration. Grouped
// declarations (`type ( A ...; B ...; )`) flatten into separate TypeDecls.
func ExtractTypes(tree *parser.Tree, src []byte) []TypeDecl {
	var out []TypeDecl
	root := tree.RootNode()
	count := int(root.NamedChildCount())
	for i := 0; i < count; i++ {
		child := root.NamedChild(i)
		if child.Type() != "type_declaration" {
			continue
		}
		for j := 0; j < int(child.NamedChildCount()); j++ {
			spec := child.NamedChild(j)
			switch spec.Type() {
			case "type_spec":
				out = append(out, decodeTypeSpec(spec, src, false))
			case "type_alias":
				out = append(out, decodeTypeSpec(spec, src, true))
			}
		}
	}
	return out
}

// decodeTypeSpec turns a single type_spec / type_alias node into a TypeDecl.
// The isAlias flag separates `type X = Y` (alias, distinct identity to Go's
// type system) from `type X struct{...}` / `type X Y` (definition).
func decodeTypeSpec(spec *sitter.Node, src []byte, isAlias bool) TypeDecl {
	decl := TypeDecl{
		Line: int(spec.StartPoint().Row) + 1,
	}
	if nameNode := spec.ChildByFieldName("name"); nameNode != nil {
		decl.Name = nameNode.Content(src)
	}
	typeNode := spec.ChildByFieldName("type")
	if typeNode == nil {
		return decl
	}
	if isAlias {
		decl.Kind = "alias"
		decl.Underlying = typeNode.Content(src)
		return decl
	}
	switch typeNode.Type() {
	case "struct_type":
		decl.Kind = "struct"
		decl.Fields = extractFields(typeNode, src)
	case "interface_type":
		decl.Kind = "interface"
		decl.Methods = extractMethods(typeNode, src)
	default:
		decl.Kind = "named"
		decl.Underlying = typeNode.Content(src)
	}
	return decl
}

// extractFields walks a struct_type's field_declaration_list and produces
// one Field per name (or one nameless Field for an embedded field).
func extractFields(structNode *sitter.Node, src []byte) []Field {
	listNode := findNamedChildOfType(structNode, "field_declaration_list")
	if listNode == nil {
		return nil
	}
	var fields []Field
	for i := 0; i < int(listNode.NamedChildCount()); i++ {
		decl := listNode.NamedChild(i)
		if decl.Type() != "field_declaration" {
			continue
		}
		typeNode := decl.ChildByFieldName("type")
		if typeNode == nil {
			continue
		}
		typeText := typeNode.Content(src)
		var names []string
		for j := 0; j < int(decl.NamedChildCount()); j++ {
			c := decl.NamedChild(j)
			if c.Type() == "field_identifier" {
				names = append(names, c.Content(src))
			}
		}
		if len(names) == 0 {
			fields = append(fields, Field{Type: typeText})
			continue
		}
		for _, n := range names {
			fields = append(fields, Field{Name: n, Type: typeText})
		}
	}
	return fields
}

// extractMethods walks an interface_type's method elements. Newer
// tree-sitter-go uses "method_elem"; older grammars used "method_spec" — we
// accept both so the analyzer survives a grammar version bump.
func extractMethods(interfaceNode *sitter.Node, src []byte) []Method {
	var methods []Method
	for i := 0; i < int(interfaceNode.NamedChildCount()); i++ {
		c := interfaceNode.NamedChild(i)
		switch c.Type() {
		case "method_elem", "method_spec":
			nameNode := c.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			methods = append(methods, Method{
				Name:        nameNode.Content(src),
				Parameters:  extractParameters(c.ChildByFieldName("parameters"), src),
				ReturnTypes: extractReturnTypes(c.ChildByFieldName("result"), src),
			})
		}
	}
	return methods
}

func findNamedChildOfType(n *sitter.Node, t string) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		if c.Type() == t {
			return c
		}
	}
	return nil
}
