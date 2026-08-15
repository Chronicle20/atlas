package rediskeyguard

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	libPkgPath     = "github.com/Chronicle20/atlas/libs/atlas-redis"
	goRedisPkgPath = "github.com/redis/go-redis/v9"
)

// bannedMethods are keyed Redis commands that take a key/field as their first
// argument. Calling any of these on the raw go-redis client/pipeliner outside
// the atlas-redis lib reintroduces the un-namespaced-key leak.
var bannedMethods = map[string]bool{
	"Set": true, "Get": true, "Del": true, "Exists": true, "Expire": true,
	"Scan": true, "Keys": true,
	"SAdd": true, "SRem": true, "SMembers": true, "SIsMember": true, "SCard": true,
	"HSet": true, "HSetNX": true, "HGet": true, "HDel": true, "HExists": true,
	"HGetAll": true, "HKeys": true, "HLen": true,
	"SetNX": true,
}

// bannedConstructors are libs/atlas-redis's env-global ("bare") type
// constructors — confirmed from source at libs/atlas-redis/registry.go:25,
// set.go:27, hash.go:16,58, keyed_set.go:20, keyed_hash.go (bare KeyedHash,
// not TenantKeyedHash), coalesced.go:64, ttl.go:23, index.go:21,74,
// id.go:21,29. Each has a Tenant-scoped sibling (NewTenantRegistry,
// NewTenantSet, NewTenantKeyedSet, NewTenantKeyedHash,
// NewTenantCoalescedRegistry, ...). Per D7, calling a bare constructor
// outside libs/atlas-redis itself reintroduces state that Redis cannot
// isolate between sparse environments sharing main's keyspace.
//
// Deliberately excluded: NewLock/NewLockWithTTL (distributed locks, not
// tenant data) and NewGlobalIDGenerator (explicitly cross-tenant by name/
// design) — these have no tenant-scoped sibling because the state they guard
// is not per-tenant.
var bannedConstructors = map[string]bool{
	"NewRegistry": true, "NewSet": true, "NewHash": true, "NewKeyedSet": true,
	"NewKeyedHash": true, "NewCoalescedRegistry": true, "NewTTLRegistry": true,
	"NewIndex": true, "NewUint32Index": true,
	"NewIDGenerator": true, "NewIDGeneratorWithStart": true,
}

// bareConstructorAllowlist names packages permitted to call a bare
// constructor, keyed by import path, with a written reason. Empty as of
// task-232/9: every fleet call site was migrated to the tenant-scoped
// equivalent rather than allowlisted. Add an entry here — never merely in a
// report or commit message — for any future bare-constructor use that is
// genuinely tenant-independent; this map is the only place the guard checks.
var bareConstructorAllowlist = map[string]string{}

var Analyzer = &analysis.Analyzer{
	Name:     "rediskeyguard",
	Doc:      "bans keyed Redis commands on the raw go-redis client, and bare (non-tenant-scoped) libs/atlas-redis constructors, outside libs/atlas-redis",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// The lib itself is the sole allowlist for the raw-client check.
	if pass.Pkg.Path() == libPkgPath {
		return nil, nil
	}
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		checkRawClientCall(pass, call)
		checkBareConstructorCall(pass, call)
	})
	return nil, nil
}

func checkRawClientCall(pass *analysis.Pass, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if !bannedMethods[sel.Sel.Name] {
		return
	}
	tv, ok := pass.TypesInfo.Types[sel.X]
	if !ok {
		return
	}
	if !isGoRedisKeyedReceiver(tv.Type) {
		return
	}
	pass.Reportf(call.Pos(),
		"rediskeyguard: %s called on raw go-redis client/pipeliner; use a libs/atlas-redis type instead",
		sel.Sel.Name)
}

func checkBareConstructorCall(pass *analysis.Pass, call *ast.CallExpr) {
	sel := selectorOf(call.Fun)
	if sel == nil {
		return
	}
	if !bannedConstructors[sel.Sel.Name] {
		return
	}
	obj := pass.TypesInfo.Uses[sel.Sel]
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != libPkgPath {
		return
	}
	if reason, allowed := bareConstructorAllowlist[pass.Pkg.Path()]; allowed {
		_ = reason
		return
	}
	pass.Reportf(call.Pos(),
		"rediskeyguard: %s is a bare (non-tenant-scoped) libs/atlas-redis constructor; use the Tenant-scoped equivalent (D7), or add this package to bareConstructorAllowlist with a written reason",
		sel.Sel.Name)
}

// selectorOf unwraps a generic instantiation (pkg.Fn[T](...) parses as an
// IndexExpr/IndexListExpr wrapping the SelectorExpr) down to the underlying
// selector, or returns nil if call.Fun isn't ultimately a selector call.
func selectorOf(fun ast.Expr) *ast.SelectorExpr {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		return f
	case *ast.IndexExpr:
		return selectorOf(f.X)
	case *ast.IndexListExpr:
		return selectorOf(f.X)
	default:
		return nil
	}
}

func isGoRedisKeyedReceiver(t types.Type) bool {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != goRedisPkgPath {
		return false
	}
	switch obj.Name() {
	case "Client", "ClusterClient", "Conn", "Pipeliner", "Tx":
		return true
	}
	return false
}
