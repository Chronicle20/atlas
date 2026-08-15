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

// eventPackageImportPrefix is the import path prefix of the concrete event
// type packages (events/crimsonbalrog, events/anniversary, ...). No package
// under event/ may import anything under this prefix — that import is the
// generic layer naming a concrete event type just as surely as a string
// literal would, and it is the more realistic violation shape: an
// IDENTIFIER or SELECTOR reference to an imported package's exported
// constant is invisible to a check that only inspects *ast.BasicLit STRING
// nodes.
const eventPackageImportPrefix = "atlas-events/events/"

// TestGenericLayerNeverNamesAnEventType walks the source of every package
// under event/ (definition, occurrence, transition, scheduling,
// orchestration, registry) with go/parser and asserts two independent things
// about FR-X3, the "generic layer never names a concrete event type" rule:
//
//  1. No non-test file contains a string literal naming a known event type
//     (knownEventTypes) — catches a bare `const` or raw-backtick literal
//     switching on a type discriminator.
//  2. No non-test file imports anything under events/... (eventPackageImportPrefix)
//     — catches a generic package importing a concrete event package and
//     naming its exported constant by IDENTIFIER or SELECTOR, which (1)
//     cannot see because that reference is not a *ast.BasicLit STRING node.
//
// What this guard does NOT catch: a type name built by concatenation or by
// fmt.Sprintf (e.g. "CRIMSON" + "_BALROG", or fmt.Sprintf("%s_BALROG",
// "CRIMSON")) is invisible to both checks — they only see literal ASTs, not
// runtime string construction. Reach event behavior through
// registry.Handler instead.
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

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(importPath, eventPackageImportPrefix) {
				t.Errorf("%s: the generic layer imports %s (FR-X3). "+
					"Reach event behavior through registry.Handler instead.",
					fset.Position(imp.Pos()), importPath)
			}
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
