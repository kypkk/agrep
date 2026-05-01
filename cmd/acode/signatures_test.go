package main

import (
	"bytes"
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
	if err := runSignatures(path, "agent", false, &buf); err != nil {
		t.Fatalf("runSignatures: %v", err)
	}
	want := "func 3 Hello(name string) error\n"
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
	if err := runSignatures(path, "human", false, &buf); err != nil {
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
	if err := runSignatures(path, "agent", false, &buf); err != nil {
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
	if err := runSignatures(path, "agent", true, &buf); err != nil {
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
	if err := runSignatures(path, "agent", false, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"User", "Name", "private"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output: %q", want, out)
		}
	}
}

func TestRunSignatures_FileNotFoundReturnsError(t *testing.T) {
	var buf bytes.Buffer
	err := runSignatures("/nonexistent/path/to/file.go", "agent", false, &buf)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRunSignatures_UnknownFormatReturnsError(t *testing.T) {
	path := writeFixture(t, "package x\n")
	var buf bytes.Buffer
	err := runSignatures(path, "json", false, &buf)
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "format") {
		t.Errorf("error should mention 'format': %v", err)
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
