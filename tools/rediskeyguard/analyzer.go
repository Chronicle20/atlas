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
// constructors that have a genuine Tenant-scoped sibling a caller could
// switch to — confirmed from source at libs/atlas-redis/registry.go:25
// (NewRegistry / NewTenantRegistry), set.go:27 (NewSet / NewTenantSet),
// hash.go:16 (NewHash — no tenant-scoped sibling exists, see below),
// keyed_set.go:20 (NewKeyedSet / NewTenantKeyedSet), hash.go:58 and
// keyed_hash.go:21 (bare KeyedHash vs TenantKeyedHash), coalesced.go:64 /
// tenant_coalesced.go:48 (NewCoalescedRegistry / NewTenantCoalescedRegistry).
// Per D7, calling a bare constructor outside libs/atlas-redis itself
// reintroduces state that Redis cannot isolate between sparse environments
// sharing main's keyspace.
//
// Deliberately excluded, and NOT in this set:
//
//   - NewLock/NewLockWithTTL (distributed locks, not tenant data) and
//     NewGlobalIDGenerator (explicitly cross-tenant by name/design) — no
//     tenant-scoped sibling exists because the state they guard is not
//     per-tenant.
//   - NewIndex/NewUint32Index (index.go:21,74), NewIDGenerator/
//     NewIDGeneratorWithStart (id.go:21,29), and NewTTLRegistry (ttl.go:23) —
//     these have a "New..." name shaped like the banned ones, but there is no
//     bare/tenant-scoped split to migrate off of: every method on Index,
//     Uint32Index and IDGenerator already takes a tenant.Model parameter and
//     keys through tenantEntityKey (index.go:29, id.go:37), and
//     NewTTLRegistry literally wraps NewTenantRegistry internally
//     (ttl.go:23-26). Flagging them is a false positive — there is no
//     tenant-scoped sibling for a caller to switch to, because they already
//     are tenant-scoped.
var bannedConstructors = map[string]bool{
	"NewRegistry": true, "NewSet": true, "NewHash": true, "NewKeyedSet": true,
	"NewKeyedHash": true, "NewCoalescedRegistry": true,
}

// bareConstructorAllowlist names packages permitted to call a bare
// constructor, keyed by import path, with a written reason. Add an entry
// here — never merely in a report or commit message — for any future
// bare-constructor use that is genuinely tenant-independent; this map is the
// only place the guard checks. Verify against source before adding: an
// allowlist entry is permanent and invisible, and this guard is the only
// thing that will ever re-examine it.
//
// Two shapes, both confirmed at task-232/9B:
//
//  1. "_tenants" enumeration indexes — a bare atlas.Set holding *which
//     tenants have data in this registry*, so a background sweep (cleanup,
//     TTL expiry, cross-tenant broadcast) can fan out over them. The set
//     itself necessarily spans every tenant sharing the environment; scoping
//     it per-tenant is incoherent — there would be nothing to enumerate.
//     Each site pairs the bare set with a Tenants()/GetTenants() accessor
//     that hands callers tenant.Model values to then scope their real,
//     per-tenant Redis calls with.
//  2. Deliberate cross-tenant indexes whose VALUES encode the tenant — the
//     set member itself is "<atlas.TenantKey(t)>:<id>", parsed back to a
//     tenant.Model by an unexported parseTenantFromKey. This is the same
//     shape as *AcrossTenants (task-232/9): the bare constructor is correct
//     because the index is deliberately cross-tenant, and every member still
//     carries its owning tenant.
var bareConstructorAllowlist = map[string]string{
	// Shape 1: "_tenants" enumeration indexes.
	"atlas-account/account": "account-session:_tenants — bare atlas.Set enumerating which " +
		"tenants have an active session registry, fanned out by Registry.Tenants(); " +
		"the set is deliberately cross-tenant by design (confirmed registry.go:60,260).",
	"atlas-buffs/berserk": "buffs:_tenants — bare atlas.Set shared with atlas-buffs/character, " +
		"enumerating tenants with active berserk state, fanned out by " +
		"Registry.GetTenants(); deliberately cross-tenant (confirmed registry.go:22,36,78).",
	"atlas-buffs/character": "buffs:_tenants — bare atlas.Set enumerating tenants with active " +
		"buff state, fanned out by Registry.GetTenants(); deliberately cross-tenant " +
		"(confirmed registry.go:39,131).",
	"atlas-character/session": "character-session:_tenants — bare atlas.Set enumerating " +
		"tenants with active character sessions; deliberately cross-tenant " +
		"(confirmed registry.go:29).",
	"atlas-expressions/expression": "expression:_tenants — bare atlas.Set enumerating tenants " +
		"with tracked expressions, fanned out by Registry.getTrackedTenants(); " +
		"deliberately cross-tenant (confirmed registry.go:30,89).",
	"atlas-invites/invite": "invite:active-tenants — bare atlas.Set enumerating tenants with " +
		"active invites, fanned out by Registry.GetActiveTenants(); deliberately " +
		"cross-tenant (confirmed registry.go:40,56).",
	"atlas-mounts/mount": "mount-active:_tenants — bare atlas.Set enumerating tenants with " +
		"active mounts; deliberately cross-tenant (confirmed registry.go:44).",
	"atlas-pets/character": "pet-character:_tenants — bare atlas.Set enumerating tenants " +
		"with tracked pet-bearing characters; deliberately cross-tenant " +
		"(confirmed registry.go:32).",
	"atlas-rps/game": "rps:_tenants — bare atlas.Set enumerating tenants with active RPS " +
		"games, fanned out by Registry.getTrackedTenants(); deliberately " +
		"cross-tenant (confirmed registry.go:34,88).",
	"atlas-skills/skill": "cooldown:_tenants — bare atlas.Set enumerating tenants with " +
		"tracked skill cooldowns; deliberately cross-tenant (confirmed " +
		"cooldown_registry.go:29-30).",
	"atlas-world/channel": "channel:tenants — bare atlas.Set enumerating tenants with " +
		"registered channels, fanned out by Registry.Tenants(); deliberately " +
		"cross-tenant (confirmed registry.go:33,79).",
	"atlas-world/broadcast": "world-broadcast:tenants — bare atlas.Set enumerating tenants " +
		"with broadcast state, fanned out by Registry.Tenants(); deliberately " +
		"cross-tenant (confirmed registry.go:34,121).",

	// Shape 2: cross-tenant indexes whose values encode the tenant.
	"atlas-drops/drop": "drops:all — bare atlas.Set whose members are " +
		"\"<atlas.TenantKey(t)>:<id>\", reconstructed via parseTenantFromKey; " +
		"deliberately cross-tenant so GetAllDrops can enumerate every drop " +
		"across tenants without an external tenant registry (confirmed " +
		"registry.go:38,55-70).",
	"atlas-reactors/reactor": "reactors:all — bare atlas.Set whose members are " +
		"\"<atlas.TenantKey(t)>:<id>\" via allSetMember/parseTenantFromKey, the same " +
		"cross-tenant-index shape as atlas-drops/drop; deliberately cross-tenant " +
		"(confirmed registry.go:42,74-76,217,248).",

	// Shape 3: hand-rolled tenant discriminator embedded in the key suffix
	// itself, rather than in a *Tenant-scoped registry's internal keying.
	"atlas-data/ingestrun": "data-ingest:<suffix> (NewJobRegistry/NewRunRegistry, both " +
		"identity-keyFn'd so every caller supplies the whole suffix) — every " +
		"caller builds that suffix through ingestrun.KeySuffix(scope, region, " +
		"major, minor) (ingestrun.go:107-109) or, for the Watchdog's Job-label " +
		"path, ingestJobKeySuffixFromLabels (runtime/rest/jobs.go:67, pinned " +
		"identical-shape by jobs_test.go:106-132) or ingestJobSuffixFromEnv " +
		"(runtime/ingest/heartbeat.go:68-82, itself routed through KeySuffix). " +
		"scope is \"tenants/<tenantId>\" or \"shared\", produced exclusively by " +
		"wzinput.ResolveScope (wzinput/scope.go:21-32): the tenant branch always " +
		"embeds the real tenant.Model.Id(), and \"shared\" is a deliberate, " +
		"operator-gated cross-tenant mode requiring the X-Atlas-Operator header, " +
		"not a default anyone reaches by accident. region/major/minor come from " +
		"the real tenant (t.Region(), t.MajorVersion(), t.MinorVersion() — " +
		"runtime/rest/resource.go:65-67,102). So the rendered suffix already " +
		"carries the same tenant+region+version discriminator libs/atlas-redis's " +
		"TenantKey renders, just with a literal \"tenants/\" prefix where TenantKey " +
		"has none; TestKeySuffixIsDiscriminating (ingestrun_test.go) pins that this " +
		"stays true. Migrating to *Tenant-scoped constructors would mean reshaping " +
		"20+ call sites across runtime/rest and runtime/ingest to arrive at a key " +
		"discriminating by the same three fields it already discriminates by, " +
		"while breaking the shared/tenant dual-mode design and the label " +
		"round-trip jobs_test.go pins — cost with no isolation benefit.",
}

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
