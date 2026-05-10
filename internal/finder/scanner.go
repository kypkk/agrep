package finder

import (
	"fmt"
	"os"

	"github.com/kypkk/agrep/internal/analyzer"
	"github.com/kypkk/agrep/internal/parser"
)

// FileResult is the parsed-and-extracted view of one .go file.
type FileResult struct {
	Path    string
	Package string
	Sigs    []analyzer.Signature
	Types   []analyzer.TypeDecl
}

// ScanFile reads, parses, and extracts a single .go file into a FileResult.
func ScanFile(path string) (FileResult, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return FileResult{}, fmt.Errorf("read %s: %w", path, err)
	}
	tree, err := parser.NewGoParser().Parse(src)
	if err != nil {
		return FileResult{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return FileResult{
		Path:    path,
		Package: analyzer.PackageName(tree, src),
		Sigs:    analyzer.ExtractSignatures(tree, src),
		Types:   analyzer.ExtractTypes(tree, src),
	}, nil
}
