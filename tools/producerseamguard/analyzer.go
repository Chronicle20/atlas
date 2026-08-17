// Package producerseamguard bans new direct producer.Produce call sites
// under services/. producer.Produce is the raw seam beneath
// producer.ProviderImpl — the canonical, composed producer (span + tenant +
// environment header decorators, task-232 FR-4.1). A direct call bypasses
// that composition and silently drops whichever decorator the seam owner
// forgot to pass by hand.
//
// The four call sites this repo already has predate the composed decorator
// and are allowlisted below; every one of them was updated in the same
// change that added producer.EnvHeaderDecorator so they carry the header
// too. Any other services/ file calling producer.Produce directly is a new
// bypass and is flagged.
package producerseamguard

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "producerseamguard",
	Doc:      "bans new direct producer.Produce calls under services/ outside the allowlisted seam call sites; compose headers through producer.ProviderImpl instead",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// producerPkgPath is the real import path of the atlas-kafka producer
// package. Matched by import path (not package name) so a local package
// that merely happens to be named "producer" is never flagged.
const producerPkgPath = "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

// allowlist holds the file paths (relative to the repo root, using "/"
// separators) of the direct producer.Produce call sites that predate this
// guard. Every entry was updated to compose producer.EnvHeaderDecorator in
// the same change that introduced this guard — see task-24 of
// task-232-sparse-ephemeral-environments/plan.md.
var allowlist = []string{
	"services/atlas-quest/atlas.com/quest/kafka/producer/quest/producer.go",
	"services/atlas-quest/atlas.com/quest/kafka/producer/saga/producer.go",
	"services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor.go",
	"services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/party_quest/processor.go",
}

func isAllowlisted(filename string) bool {
	norm := filepathToSlash(filename)
	for _, a := range allowlist {
		if strings.HasSuffix(norm, a) {
			return true
		}
	}
	return false
}

func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Produce" {
			return
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return
		}
		pkgName, ok := pass.TypesInfo.Uses[ident].(*types.PkgName)
		if !ok || pkgName.Imported().Path() != producerPkgPath {
			return
		}

		filename := pass.Fset.Position(call.Pos()).Filename
		if isAllowlisted(filename) {
			return
		}
		pass.Reportf(call.Pos(), "direct producer.Produce outside libs/; compose headers through producer.ProviderImpl or add this call site to tools/producerseamguard's allowlist with a written reason")
	})
	return nil, nil
}
