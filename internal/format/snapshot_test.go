package format

import (
	"testing"

	"github.com/kypkk/acode/internal/analyzer"
	"github.com/kypkk/acode/internal/parser"
)

// TestSnapshot_Visual is a visual smoke test — runs the formatters on a
// synthetic Go file and logs the output. Run with `go test -v -run Snapshot`
// to inspect.
func TestSnapshot_Visual(t *testing.T) {
	src := []byte(`package demo

// Hello greets the world.
// It returns nothing meaningful.
func Hello(name string, times int) error { return nil }

// User is a customer record.
type User struct {
	Name, Email string
	Age         int
}

type Reader interface {
	Read(p []byte) (n int, err error)
	Close() error
}

type Names = []string
type Counter int
`)
	p := parser.NewGoParser()
	tree, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	sigs := analyzer.ExtractSignatures(tree, src)
	types := analyzer.ExtractTypes(tree, src)

	t.Logf("=== AGENT ===\n%s", Agent(sigs, types))
	t.Logf("=== HUMAN (no color) ===\n%s", Human(sigs, types, HumanOptions{Color: false}))
}
