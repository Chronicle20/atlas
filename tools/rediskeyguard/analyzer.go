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

var Analyzer = &analysis.Analyzer{
	Name:     "rediskeyguard",
	Doc:      "bans keyed Redis commands on the raw go-redis client outside libs/atlas-redis",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// The lib itself is the sole allowlist.
	if pass.Pkg.Path() == libPkgPath {
		return nil, nil
	}
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
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
	})
	return nil, nil
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
