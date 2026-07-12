package outboxguard

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:      "outboxguard",
	Doc:       "bans direct Kafka producer construction (producer.ProviderImpl) inside DB transaction closures, including through concrete (statically-resolvable) helper calls",
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	Run:       run,
	FactTypes: []analysis.Fact{(*emitsDirectFact)(nil)},
}

// emitsDirectFact marks a function/method whose body, when called
// synchronously, constructs the direct Kafka producer (producer.ProviderImpl) —
// either lexically or transitively through a statically-resolvable (concrete)
// call. It is exported per function object so a dependent package's transaction
// closure can be flagged for calling such a helper.
//
// Two deliberate boundaries keep this a sound tripwire rather than a whole-
// program taint pass:
//   - Interface-method calls are NOT followed. An interface is the dependency-
//     injection seam (e.g. a SagaEmitter that at the call site may be a
//     tx-bound outbox emitter, per the atlas-mts escrow fix); a concrete
//     producer.ProviderImpl reached only through an interface is out of scope
//     and remains covered by review + the documented outbox pattern.
//   - A producer.ProviderImpl referenced only inside a NESTED func literal is
//     treated as deferred/post-transaction (the rejectEmit pattern) and does
//     not, by itself, mark the enclosing function.
type emitsDirectFact struct{}

func (*emitsDirectFact) AFact()         {}
func (*emitsDirectFact) String() string { return "emitsDirect" }

func run(pass *analysis.Pass) (interface{}, error) {
	// --- Phase 1: compute and export per-function "emits direct" facts. ---
	decls := map[*types.Func]*ast.FuncDecl{}
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			if obj, ok := pass.TypesInfo.Defs[fd.Name].(*types.Func); ok {
				decls[obj] = fd
			}
		}
	}

	emits := map[*types.Func]bool{}
	callees := map[*types.Func][]*types.Func{}
	for obj, fd := range decls {
		if lexicalDirectEmit(pass, fd.Body) {
			emits[obj] = true
		}
		callees[obj] = concreteCallees(pass, fd.Body)
	}

	// Local fixed point: a function emits if it lexically does, or calls a
	// concrete callee that emits (a local one computed here, or an imported one
	// carrying the fact).
	for changed := true; changed; {
		changed = false
		for obj, cs := range callees {
			if emits[obj] {
				continue
			}
			for _, c := range cs {
				if emits[c] || importedEmits(pass, c) {
					emits[obj] = true
					changed = true
					break
				}
			}
		}
	}
	for obj := range emits {
		pass.ExportObjectFact(obj, &emitsDirectFact{})
	}

	// --- Phase 2: flag direct emits (and calls to emitting helpers) that run
	// inside a DB transaction closure. ---
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
				// A nested func literal (e.g. a deferred rejectEmit closure) runs
				// strictly after the transaction returns; don't descend into it.
				if innerFl, ok := inner.(*ast.FuncLit); ok && innerFl != fl {
					return false
				}
				// (a) A direct producer.ProviderImpl in the tx control flow.
				if sel, ok := inner.(*ast.SelectorExpr); ok && isProviderImpl(pass, sel) {
					pass.Reportf(sel.Pos(),
						"outboxguard: producer.ProviderImpl inside a DB transaction closure; enqueue via outbox.EmitProvider instead")
					return true
				}
				// (b) A call to a concrete helper that transitively reaches a
				// direct producer.ProviderImpl.
				if c, ok := inner.(*ast.CallExpr); ok {
					if callee := calleeFunc(pass, c); callee != nil && (emits[callee] || importedEmits(pass, callee)) {
						pass.Reportf(c.Pos(),
							"outboxguard: %s reaches a direct producer.ProviderImpl and runs inside a DB transaction closure; enqueue via outbox.EmitProvider or move the emit after the transaction",
							callee.Name())
					}
				}
				return true
			})
		}
	})
	return nil, nil
}

// lexicalDirectEmit reports whether body references producer.ProviderImpl in its
// own control flow — i.e. outside any nested func literal (which is treated as
// deferred/post-tx, matching the check's nested-closure exclusion).
func lexicalDirectEmit(pass *analysis.Pass, body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		if _, ok := n.(*ast.FuncLit); ok {
			return false // don't descend into nested closures
		}
		if sel, ok := n.(*ast.SelectorExpr); ok && isProviderImpl(pass, sel) {
			found = true
			return false
		}
		return true
	})
	return found
}

// concreteCallees returns the statically-resolvable callee functions invoked in
// body's own control flow (outside nested func literals). Interface-method calls
// are excluded — they are the dependency-injection seam.
func concreteCallees(pass *analysis.Pass, body *ast.BlockStmt) []*types.Func {
	var out []*types.Func
	seen := map[*types.Func]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if c, ok := n.(*ast.CallExpr); ok {
			if f := calleeFunc(pass, c); f != nil && !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
		return true
	})
	return out
}

// calleeFunc resolves a call to the concrete *types.Func it invokes, or nil when
// the callee is not a statically-resolvable function (an interface method, a
// func value, a builtin, or a conversion).
func calleeFunc(pass *analysis.Pass, call *ast.CallExpr) *types.Func {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		// package-local function call: f(...)
		if fn, ok := pass.TypesInfo.Uses[fun].(*types.Func); ok {
			return fn
		}
	case *ast.SelectorExpr:
		// Method value or qualified function. A selection with an interface
		// receiver is dependency-injected — do not follow it.
		if selc, ok := pass.TypesInfo.Selections[fun]; ok {
			if selc.Kind() == types.MethodVal {
				if recv := selc.Recv(); recv != nil && types.IsInterface(recv) {
					return nil
				}
				if fn, ok := selc.Obj().(*types.Func); ok {
					return fn
				}
			}
			return nil
		}
		// pkg.Func(...) — a package-qualified function, not a method selection.
		if fn, ok := pass.TypesInfo.Uses[fun.Sel].(*types.Func); ok {
			if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() == nil {
				return fn
			}
		}
	}
	return nil
}

// importedEmits reports whether an (imported) callee carries the emitsDirect
// fact exported by its defining package.
func importedEmits(pass *analysis.Pass, fn *types.Func) bool {
	var fact emitsDirectFact
	return pass.ImportObjectFact(fn, &fact)
}

// isProviderImpl reports whether sel is a reference to the service-local
// kafka/producer package's ProviderImpl constructor.
func isProviderImpl(pass *analysis.Pass, sel *ast.SelectorExpr) bool {
	if sel.Sel.Name != "ProviderImpl" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkgName, ok := pass.TypesInfo.Uses[ident].(*types.PkgName)
	return ok && pkgName.Imported().Name() == "producer"
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
