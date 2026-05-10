package finder

import (
	"path/filepath"
	"testing"
)

func TestScanFile_ExtractsPackageSigsAndTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.go")
	mustWrite(t, path, `package mypkg

func Hello() error { return nil }

type User struct {
	Name string
}
`)
	got, err := ScanFile(path)
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if got.Path != path {
		t.Errorf("Path = %q, want %q", got.Path, path)
	}
	if got.Package != "mypkg" {
		t.Errorf("Package = %q, want mypkg", got.Package)
	}
	if len(got.Sigs) != 1 || got.Sigs[0].Name != "Hello" {
		t.Errorf("Sigs = %+v, want one Hello signature", got.Sigs)
	}
	if len(got.Types) != 1 || got.Types[0].Name != "User" {
		t.Errorf("Types = %+v, want one User type", got.Types)
	}
}

func TestScanFile_NonexistentReturnsError(t *testing.T) {
	if _, err := ScanFile("/no/such/file.go"); err == nil {
		t.Error("expected error for missing file")
	}
}
