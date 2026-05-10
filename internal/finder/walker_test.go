package finder

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestWalk_DirectoryRecursive(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.go"), "package x\n")
	mustWrite(t, filepath.Join(root, "sub/b.go"), "package x\n")
	mustWrite(t, filepath.Join(root, "sub/c.txt"), "ignore me")
	mustWrite(t, filepath.Join(root, "vendor/v.go"), "package v\n")
	mustWrite(t, filepath.Join(root, ".hidden/h.go"), "package h\n")
	mustWrite(t, filepath.Join(root, "_internal/i.go"), "package i\n")
	mustWrite(t, filepath.Join(root, "node_modules/n.go"), "package n\n")

	got, err := Walk(root)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	want := []string{
		filepath.Join(root, "a.go"),
		filepath.Join(root, "sub/b.go"),
	}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("got %d files, want %d: %v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWalk_SingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lone.go")
	mustWrite(t, path, "package x\n")

	got, err := Walk(path)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(got) != 1 || got[0] != path {
		t.Errorf("got %v, want [%s]", got, path)
	}
}

func TestWalk_NonexistentReturnsError(t *testing.T) {
	if _, err := Walk("/no/such/path/here"); err == nil {
		t.Error("expected error for missing path")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
