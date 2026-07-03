package outboxguard

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "outboxguard",
	Doc:      "bans direct Kafka producer construction (producer.ProviderImpl) inside DB transaction closures",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// The guard is a lexical regression tripwire, not a taint analysis: the
// fleet's only direct-producer entry point in service code is the local
// kafka/producer.ProviderImpl, and transaction entry points are uniformly
// database.ExecuteTransaction or gorm's (*DB).Transaction.
func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		if !isTxEntryPoint(call) {
			return
		}
		for _, arg := range call.Args {
			fl, ok := arg.(*ast.FuncLit)
			if !ok {
				continue
			}
			ast.Inspect(fl.Body, func(inner ast.Node) bool {
				if innerFl, ok := inner.(*ast.FuncLit); ok && innerFl != fl {
					// A nested func literal (e.g. a deferred rejectEmit
					// closure) may capture producer.ProviderImpl for
					// invocation strictly after the transaction returns.
					// That's not in-tx execution, so don't descend into it.
					return false
				}
				sel, ok := inner.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "ProviderImpl" {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				pkgName, ok := pass.TypesInfo.Uses[ident].(*types.PkgName)
				if !ok || pkgName.Imported().Name() != "producer" {
					return true
				}
				pass.Reportf(sel.Pos(),
					"outboxguard: producer.ProviderImpl inside a DB transaction closure; enqueue via outbox.EmitProvider instead")
				return true
			})
		}
	})
	return nil, nil
}

func isTxEntryPoint(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "ExecuteTransaction":
		ident, ok := sel.X.(*ast.Ident)
		return ok && ident.Name == "database"
	case "Transaction":
		return true
	}
	return false
}
