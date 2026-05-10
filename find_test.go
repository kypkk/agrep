package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFindFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRunFind_HasMethodGroupsByType(t *testing.T) {
	root := writeFindFixture(t, map[string]string{
		"order.go": `package svc

type OrderService struct{}

func (s *OrderService) Delete(id string) error { return nil }
`,
		"product.go": `package svc

type ProductService struct{}

func (s *ProductService) Delete(id string) error { return nil }
`,
		"user.go": `package svc

type UserService struct{}

func (s *UserService) Update(id string) error { return nil }
`,
	})
	var stdout, stderr bytes.Buffer
	if err := runFind(FindOptions{In: root, HasMethod: "Delete"}, &stdout, &stderr); err != nil {
		t.Fatalf("runFind: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"OrderService", "ProductService", "Delete", "2 results"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
	if strings.Contains(out, "UserService") {
		t.Errorf("UserService should not match --has-method=Delete:\n%s", out)
	}
}

func TestRunFind_ReturnsFlatList(t *testing.T) {
	root := writeFindFixture(t, map[string]string{
		"a.go": `package u

func Marshal(v any) (string, error) { return "", nil }

func Unmarshal(s string, v any) error { return nil }

func ReadFile(p string) ([]byte, error) { return nil, nil }
`,
	})
	var stdout, stderr bytes.Buffer
	if err := runFind(FindOptions{In: root, Returns: "error"}, &stdout, &stderr); err != nil {
		t.Fatalf("runFind: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"Marshal", "Unmarshal", "ReadFile", "3 results"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestRunFind_NoResultsShowsSuggestions(t *testing.T) {
	root := writeFindFixture(t, map[string]string{
		"a.go": `package svc

type X struct{}

func (s *X) Create() error { return nil }

func (s *X) Update() error { return nil }
`,
	})
	var stdout, stderr bytes.Buffer
	if err := runFind(FindOptions{In: root, HasMethod: "Save"}, &stdout, &stderr); err != nil {
		t.Fatalf("runFind: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "No results") {
		t.Errorf("expected `No results` in output: %s", out)
	}
	if !strings.Contains(out, "Did you mean") {
		t.Errorf("expected `Did you mean` in output: %s", out)
	}
}

func TestRunFind_EmptyFilterReturnsError(t *testing.T) {
	root := writeFindFixture(t, map[string]string{"a.go": "package x\n"})
	err := runFind(FindOptions{In: root}, io.Discard, io.Discard)
	if err == nil {
		t.Error("expected error for empty filter")
	}
}

func TestRunFind_InvalidKindReturnsError(t *testing.T) {
	root := writeFindFixture(t, map[string]string{"a.go": "package x\n"})
	err := runFind(FindOptions{In: root, Kind: "garbage"}, io.Discard, io.Discard)
	if err == nil {
		t.Error("expected error for invalid kind")
	}
}

func TestRunFind_PaginationFooter(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 30; i++ {
		fn := nameFn(i)
		files[fn+".go"] = "package x\nfunc " + fn + "() error { return nil }\n"
	}
	root := writeFindFixture(t, files)
	var stdout, stderr bytes.Buffer
	if err := runFind(FindOptions{In: root, Returns: "error", Limit: 5}, &stdout, &stderr); err != nil {
		t.Fatalf("runFind: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "use --all to see all") {
		t.Errorf("expected pagination footer, got:\n%s", out)
	}
}

func nameFn(i int) string {
	return "F" + string(rune('A'+i%26)) + string(rune('a'+(i/26)%26))
}
