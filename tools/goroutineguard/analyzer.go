package goroutineguard

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	// The helper lib is the only package allowed to contain bare go statements.
	routinePkgPath = "github.com/Chronicle20/atlas/libs/atlas-routine"
	markerPrefix   = "//goroutine-guard:allow"
)

var Analyzer = &analysis.Analyzer{
	Name:     "goroutineguard",
	Doc:      "bans bare go statements outside libs/atlas-routine; spawn via routine.Go instead",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

type lineKey struct {
	file string
	line int
}

func run(pass *analysis.Pass) (interface{}, error) {
	if strings.HasPrefix(pass.Pkg.Path(), routinePkgPath) {
		return nil, nil
	}

	// Collect allow markers: file:line → justification present?
	markers := map[lineKey]bool{}
	for _, f := range pass.Files {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if !strings.HasPrefix(c.Text, markerPrefix) {
					continue
				}
				pos := pass.Fset.Position(c.Pos())
				justification := strings.TrimSpace(strings.TrimPrefix(c.Text, markerPrefix))
				markers[lineKey{pos.Filename, pos.Line}] = justification != ""
			}
		}
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.GoStmt)(nil)}, func(n ast.Node) {
		pos := pass.Fset.Position(n.Pos())
		if strings.HasSuffix(pos.Filename, "_test.go") {
			return
		}
		if justified, found := markerFor(markers, pos); found {
			if !justified {
				pass.Reportf(n.Pos(), "goroutineguard: allow marker requires a justification")
			}
			return
		}
		pass.Reportf(n.Pos(), "goroutineguard: bare go statement; use routine.Go from libs/atlas-routine (or add //goroutine-guard:allow <justification>)")
	})
	return nil, nil
}

// markerFor accepts a marker trailing on the statement's own line or on the
// line immediately above it.
func markerFor(markers map[lineKey]bool, pos token.Position) (justified bool, found bool) {
	if justified, found = markers[lineKey{pos.Filename, pos.Line}]; found {
		return justified, found
	}
	justified, found = markers[lineKey{pos.Filename, pos.Line - 1}]
	return justified, found
}
