package event

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// knownEventTypes are the type discriminators an event package owns. The
// GENERIC layer must never name one: FR-X3 permits "a registry mapping type to
// a handler" and forbids "a switch containing event logic". A literal here in
// event/definition, event/occurrence, event/transition, event/scheduling,
// event/orchestration or event/registry is that forbidden switch beginning to
// form.
var knownEventTypes = []string{"CRIMSON_BALROG", "ANNIVERSARY"}

// minInspectedFiles is a sanity floor on how many .go files the walk visits.
// If the walk root were wrong (e.g. run from a directory that resolves to
// something other than event/), the test would pass by inspecting zero
// files — the same false green R39-2 describes, in a different disguise.
// The generic layer has 30+ non-test .go files today across six packages;
// 10 is comfortably below that while still catching "the walk saw nothing."
const minInspectedFiles = 10

// TestGenericLayerNeverNamesAnEventType walks the source of every package
// under event/ (definition, occurrence, transition, scheduling,
// orchestration, registry) with go/parser and fails if any non-test file
// contains a string literal naming a known event type. Reach event behavior
// through registry.Handler instead.
func TestGenericLayerNeverNamesAnEventType(t *testing.T) {
	root := "."
	inspected := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		inspected++
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}
		ast.Inspect(f, func(n ast.Node) bool {
			bl, ok := n.(*ast.BasicLit)
			if !ok || bl.Kind != token.STRING {
				return true
			}
			for _, known := range knownEventTypes {
				if strings.Contains(bl.Value, known) {
					t.Errorf("%s: the generic layer names event type %s (FR-X3). "+
						"Reach event behavior through registry.Handler instead.",
						fset.Position(bl.Pos()), known)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if inspected < minInspectedFiles {
		t.Fatalf("walk inspected only %d .go files under %q (want >= %d) — "+
			"the walk root is probably wrong and this test is passing by "+
			"seeing nothing", inspected, root, minInspectedFiles)
	}
}
