package finder

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Walk returns the .go files reachable from root. root may be a single .go
// file (returned as-is) or a directory (walked recursively, skipping
// vendor/, node_modules/, and any directory whose name starts with "." or "_").
func Walk(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}
	if !info.IsDir() {
		if !strings.HasSuffix(root, ".go") {
			return nil, fmt.Errorf("%s is not a .go file", root)
		}
		return []string{root}, nil
	}

	var paths []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(p, root, d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func shouldSkipDir(path, root, name string) bool {
	if path == root {
		return false // never skip the root itself
	}
	switch name {
	case "vendor", "node_modules":
		return true
	}
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}
