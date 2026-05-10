package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.go")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestRunSignatures_AgentFormatExactOutput(t *testing.T) {
	path := writeFixture(t, `package x

func Hello(name string) error { return nil }
`)
	var buf bytes.Buffer
	if err := runSignatures(path, "agent", false, &buf, io.Discard); err != nil {
		t.Fatalf("runSignatures: %v", err)
	}
	// The agent output is prefixed with `file:` and `package:` lines so the
	// consumer always knows where the rest came from. The fixture path is
	// dynamic (t.TempDir), so format the expected output with it.
	want := fmt.Sprintf("file: %s\npackage: x\nfunc 3 Hello(name string) error\n", path)
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunSignatures_HumanFormatContainsKeyParts(t *testing.T) {
	path := writeFixture(t, `package x

// Hello greets you.
func Hello() {}
`)
	var buf bytes.Buffer
	if err := runSignatures(path, "human", false, &buf, io.Discard); err != nil {
		t.Fatalf("runSignatures: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Hello", "Hello greets you."} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output: %q", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("human output should have no ANSI when writer is not a TTY: %q", out)
	}
}

func TestRunSignatures_FilterUnexportedByDefault(t *testing.T) {
	path := writeFixture(t, `package x

func PublicFunc() {}
func privateFunc() {}

type PublicType struct{}

type privateType struct{}
`)
	var buf bytes.Buffer
	if err := runSignatures(path, "agent", false, &buf, io.Discard); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"PublicFunc", "PublicType"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output: %q", want, out)
		}
	}
	for _, hidden := range []string{"privateFunc", "privateType"} {
		if strings.Contains(out, hidden) {
			t.Errorf("did not expect %q without --all: %q", hidden, out)
		}
	}
}

func TestRunSignatures_AllIncludesUnexported(t *testing.T) {
	path := writeFixture(t, `package x

func privateFunc() {}

type privateType struct{}
`)
	var buf bytes.Buffer
	if err := runSignatures(path, "agent", true, &buf, io.Discard); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"privateFunc", "privateType"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q with --all: %q", want, out)
		}
	}
}

func TestRunSignatures_KeepsAllMembersOfIncludedType(t *testing.T) {
	// Even with --all=false, members of an included exported type are
	// preserved as-is — matches `go doc` behavior.
	path := writeFixture(t, `package x

type User struct {
	Name    string
	private int
}
`)
	var buf bytes.Buffer
	if err := runSignatures(path, "agent", false, &buf, io.Discard); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"User", "Name", "private"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output: %q", want, out)
		}
	}
}

func TestRunSignatures_JSONFormat(t *testing.T) {
	path := writeFixture(t, `package mypkg

// Hello greets you.
func Hello() error { return nil }

type User struct {
	Name string
}
`)
	var buf bytes.Buffer
	if err := runSignatures(path, "json", false, &buf, io.Discard); err != nil {
		t.Fatalf("runSignatures: %v", err)
	}
	out := buf.String()
	// Valid JSON
	var parsed struct {
		File      string `json:"file"`
		Package   string `json:"package"`
		Functions []struct {
			Name     string `json:"name"`
			Doc      string `json:"doc"`
			Exported bool   `json:"exported"`
		} `json:"functions"`
		Types []struct {
			Name   string `json:"name"`
			Kind   string `json:"kind"`
			Fields []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"fields"`
		} `json:"types"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if parsed.Package != "mypkg" {
		t.Errorf("package = %q, want mypkg", parsed.Package)
	}
	if !strings.HasSuffix(parsed.File, "input.go") {
		t.Errorf("file = %q, want suffix input.go", parsed.File)
	}
	if len(parsed.Functions) != 1 || parsed.Functions[0].Name != "Hello" {
		t.Errorf("functions: %+v", parsed.Functions)
	}
	if parsed.Functions[0].Doc != "Hello greets you." {
		t.Errorf("doc = %q", parsed.Functions[0].Doc)
	}
	if len(parsed.Types) != 1 || parsed.Types[0].Kind != "struct" {
		t.Errorf("types: %+v", parsed.Types)
	}
	if len(parsed.Types[0].Fields) != 1 || parsed.Types[0].Fields[0].Name != "Name" {
		t.Errorf("fields: %+v", parsed.Types[0].Fields)
	}
}

func TestRunSignatures_FileNotFoundReturnsError(t *testing.T) {
	var buf bytes.Buffer
	err := runSignatures("/nonexistent/path/to/file.go", "agent", false, &buf, io.Discard)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRunSignatures_UnknownFormatReturnsError(t *testing.T) {
	path := writeFixture(t, "package x\n")
	var buf bytes.Buffer
	err := runSignatures(path, "yaml", false, &buf, io.Discard)
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "format") {
		t.Errorf("error should mention 'format': %v", err)
	}
}

func TestRunSignatures_HintWhenFilterEmptiesOutput(t *testing.T) {
	path := writeFixture(t, `package main

func main() {}

func walk() {}
`)
	var stdout, stderr bytes.Buffer
	if err := runSignatures(path, "human", false, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	// stdout still carries the file/package header; the body should be empty
	// since every entry was filtered. We don't assert "stdout is empty" — the
	// header is desirable. The contract this test pins is "the hint appears
	// on stderr when the filter swallows everything".
	if strings.Contains(stdout.String(), "func ") || strings.Contains(stdout.String(), "method ") {
		t.Errorf("stdout body should be empty (all decls filtered), got: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--all") {
		t.Errorf("stderr should hint about --all, got: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "2") {
		t.Errorf("stderr should mention 2 dropped declarations, got: %q", stderr.String())
	}
}

func TestRunSignatures_HintFiresAcrossAllFormats(t *testing.T) {
	// JSON output is technically non-empty (`{..., "functions": [], "types": []}`)
	// but the underlying data is empty — the hint should still fire.
	path := writeFixture(t, "package x\nfunc lower() {}\n")
	for _, fmt := range []string{"human", "agent", "json"} {
		t.Run(fmt, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := runSignatures(path, fmt, false, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stderr.String(), "--all") {
				t.Errorf("[%s] expected --all hint on stderr, got: %q", fmt, stderr.String())
			}
		})
	}
}

func TestRunSignatures_NoHintWhenAllFlagAlreadyTrue(t *testing.T) {
	path := writeFixture(t, "package x\nfunc lower() {}\n")
	var stdout, stderr bytes.Buffer
	if err := runSignatures(path, "agent", true, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty when --all=true, got: %q", stderr.String())
	}
}

func TestRunSignatures_NoHintWhenOutputHasContent(t *testing.T) {
	path := writeFixture(t, `package x

func Public() {}

func private() {}
`)
	var stdout, stderr bytes.Buffer
	if err := runSignatures(path, "agent", false, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty when at least one decl survives, got: %q", stderr.String())
	}
}

func TestRunSignatures_NoHintForGenuinelyEmptyFile(t *testing.T) {
	// Zero declarations to begin with — the hint would be misleading.
	path := writeFixture(t, "package x\n")
	var stdout, stderr bytes.Buffer
	if err := runSignatures(path, "agent", false, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty for file with no declarations, got: %q", stderr.String())
	}
}

func TestIsExportedName(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Foo", true},
		{"foo", false},
		{"", false},
		{"_internal", false},
		{"Ångström", true},
	}
	for _, c := range cases {
		if got := isExportedName(c.in); got != c.want {
			t.Errorf("isExportedName(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
