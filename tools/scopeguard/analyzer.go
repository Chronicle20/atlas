// Package scopeguard implements FR-8.5: a `go vet`-driven analyzer that
// keeps the query-scope audit (docs/tasks/task-232-sparse-ephemeral-environments/
// query-scope-audit.md) from rotting while Phases B-F of task-232 are still
// in flight.
//
// It enforces two independent rules, at two different granularities:
//
//  1. Entity-level (per file): a `services/**/entity.go`-shaped file's
//     primary `Entity` struct must carry a scoping column — `TenantId` for a
//     data-plane entity, `Environment` for a control-plane one (the two
//     services that host the tenant/environment registries themselves:
//     atlas-configurations, atlas-tenants). Either may be excused, but only
//     for the "the row IS the scoping dimension" shape (a data-plane entity
//     that IS the tenant, or the one control-plane entity that IS the
//     environment list itself — atlas-configurations/environments.Entity,
//     task-232 Task 19), and only when THREE independent conditions all
//     hold at once:
//
//     - a reason in allowlist.txt, keyed off the declaring file
//     (entityAllowlistKey) — prose, reviewed once at the line's own
//     addition;
//     - hasUniqueNaturalKey(st) — the struct declares a non-surrogate
//     field with a real `gorm:"...uniqueIndex..."` DB constraint;
//     - hasScopingDimensionMarker — the entity itself declares a
//     `func (Entity) ScopingDimension() {}` marker method, so the claim
//     is stated at the entity's own declaration site, not just in a
//     text file far from the code.
//
//     No two of the three are sufficient. Fix round 1 (Task 19) proved an
//     allowlist-only gate is smuggleable: any entity author can add a
//     prose-justified line for an entity that is not actually the scoping
//     dimension (the `auditrow` fixture). Fix round 2 proved
//     hasUniqueNaturalKey alone is ALSO just a proxy: a control-plane row
//     that is emphatically not the scoping dimension (an idempotency-keyed
//     audit/job record) can carry an unrelated unique key and pass the
//     structural check anyway (the `smuggle2` fixture). Struct shape cannot
//     reliably decide "is this the top-level enumeration" — that is a
//     semantic claim, not a syntactic property, and each shape refinement
//     is one clever entity away from the next bypass. The marker method
//     requires the author to STATE the claim in code, where a reviewer
//     reading the entity's diff will see it and `grep -r
//     ScopingDimension` finds every instance fleet-wide. An entity that
//     merely forgot its scoping column, or fails any one of the three
//     checks, is not excusable.
//
//  2. Call-site-level (per call, fleet-wide over services/ AND libs/): the
//     two unscoping mechanisms the audit's §3 "second-mechanism sweep"
//     found, confirmed at `libs/atlas-database/tenant_scope.go`:
//
//     - explicit opt-out: any call to `...WithoutTenantFilter(...)`, the
//     one function that turns off the fleet-wide GORM tenant callback,
//     outside libs/atlas-database itself;
//     - the silent-no-op path: a GORM query verb invoked on a struct field
//     literally named `db` or `DB` whose call chain never reaches a
//     `.WithContext(` — which collapses the callback's context lookup to
//     `context.Background()` and it silently skips the tenant filter
//     (tenant_scope.go:60,69-73). This is the exact shape that let
//     atlas-marriages' two schedulers evade the first (per-service
//     aggregate) sweep: `s.db.Model(...).Where(...).Pluck(...)` sits next
//     to correctly-wrapped `provider.go` call sites in the same service.
//
// Both call-site findings may be excused via callsite-allowlist.txt, keyed
// by package import path + call site line, with a written reason — the
// fleet currently carries thirteen genuine INTENDED-GLOBAL background-sweep
// call sites (§4 of the audit), each with its own source-comment evidence.
package scopeguard

import (
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// controlPlaneServices host the tenant/environment registries themselves —
// their primary role is to enumerate tenants/environments, not to be
// per-tenant data. Confirmed at query-scope-audit.md rows 72-74, 144-145.
var controlPlaneServices = map[string]bool{
	"atlas-configurations": true,
	"atlas-tenants":        true,
}

// gormVerbs are the GORM query-builder methods the audit's §3 struct-field
// scan grepped for (tenant_scope.go's callback only fires on these).
var gormVerbs = map[string]bool{
	"Model": true, "Where": true, "Find": true, "First": true,
	"Create": true, "Save": true, "Delete": true, "Updates": true,
	"Update": true, "Pluck": true, "Count": true, "Exec": true, "Raw": true,
	"FirstOrCreate": true,
}

const libDatabasePkgSuffix = "libs/atlas-database"

var Analyzer = &analysis.Analyzer{
	Name:     "scopeguard",
	Doc:      "FR-8.5: flags entity.go structs missing their scoping column, and call sites that bypass the fleet-wide tenant-scope GORM callback",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	insp.Preorder([]ast.Node{(*ast.TypeSpec)(nil)}, func(n ast.Node) {
		checkEntity(pass, n.(*ast.TypeSpec))
	})

	if !strings.HasSuffix(pass.Pkg.Path(), libDatabasePkgSuffix) {
		checkCallSites(pass, insp)
	}

	return nil, nil
}

// --- Rule 1: entity-level ---

func checkEntity(pass *analysis.Pass, ts *ast.TypeSpec) {
	if !isEntityTypeName(ts.Name.Name) {
		return
	}
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return
	}

	svc, ok := serviceFromPkgPath(pass.Pkg.Path())
	if !ok {
		// Not a services/atlas-<name> module at all (libs/, tools/) — the
		// entity-level rule is scoped to services/**/entity.go per the
		// brief; skip.
		return
	}

	// Both spellings are live in the fleet: most services use `TenantId`,
	// but atlas-map-actions/npc-conversations/party-quests/portal-actions/
	// reactor-actions use the Go-initialism-correct `TenantID` (confirmed at
	// e.g. services/atlas-map-actions/.../script/entity.go:17 — both map to
	// the same `tenant_id` DB column the callback matches on by column name,
	// not Go field name; tenant_scope.go:31-37).
	if hasField(st, "TenantId") || hasField(st, "TenantID") {
		// Scoped by the fleet-wide GORM callback (tenant_scope.go:75-79) —
		// data-plane, correctly scoped, regardless of service.
		return
	}

	key := entityAllowlistKey(pass, ts.Pos())

	if controlPlaneServices[svc] {
		if hasField(st, "Environment") {
			return
		}
		// A control-plane entity that IS the environment list itself (e.g.
		// atlas-configurations/environments.Entity) has nothing above
		// "environment" to scope by — the same shape as a data-plane entity
		// that IS the tenant and so carries no TenantId. That is the one
		// allowlist-able exception to Rule 1's control-plane branch, and it
		// is gated on THREE independent conditions, all required (task-232
		// Task 19, fix rounds 1-3):
		//   - an allowlist entry: prose an entity author controls, so not
		//     sufficient alone — fix round 1 proved that by smuggling a
		//     plain audit-row entity (no scoping field, no natural key)
		//     past an allowlist-only version of this check;
		//   - hasUniqueNaturalKey: also just a proxy, not sufficient alone
		//     — fix round 2 proved that by smuggling an idempotency-keyed
		//     audit row (an unrelated uniqueIndex column that has nothing
		//     to do with being the scoping dimension) past an
		//     allowlist+shape-only version;
		//   - hasScopingDimensionMarker: the entity must declare
		//     `func (Entity) ScopingDimension() {}` on itself, so the claim
		//     is stated at the entity's own site, not inferred from shape.
		// Every other control-plane entity without Environment is a real
		// gap and stays hard-denied regardless of what allowlist.txt says
		// or what fields it carries (see "atlas-configurations/thing" and
		// the pinned smuggle-probe fixtures — auditrow, testfileaudit,
		// smuggle2 — in analyzer_test.go).
		if reason, ok := EntityAllowlist[key]; ok && hasUniqueNaturalKey(st) && hasScopingDimensionMarker(pass, ts.Name.Name) {
			_ = reason
			return
		}
		pass.Reportf(ts.Pos(), "control-plane entity without Environment")
		return
	}

	if reason, ok := EntityAllowlist[key]; ok {
		_ = reason
		return
	}
	pass.Reportf(ts.Pos(), "data-plane entity without TenantId")
}

// isEntityTypeName reports whether a struct's type name matches Rule 1's
// entity-shape convention. The fleet carries three live spellings, confirmed
// by grep across services/ (query-scope-audit.md follow-up, fix round 2):
//
//   - the plain, exact `Entity` — the majority convention;
//   - the plain, exact lowercase `entity` — atlas-monster-book/card,
//     atlas-reward-pools/item, atlas-cashshop/surprise/opening,
//     atlas-drop-information (reactor/drop, continent/drop, monster/drop),
//     atlas-monster-book/collection, atlas-reward-pools/gachapon,
//     atlas-keys/key, atlas-maps/character/location;
//   - an `Entity`-suffixed PascalCase name — ItemEntity/MesoEntity (atlas-
//     trades/escrow), HistoryEntity (atlas-configurations), CycleEntity
//     (atlas-rankings/ranking), SearchIndexEntity (atlas-data reactor/npc/
//     monster/map).
//
// Deliberately excluded: a bare `Entity`-suffix match on the lowercase form
// (e.g. a hypothetical `itementity`) — no such name exists fleet-wide, and
// matching it would risk flagging an unrelated struct that merely ends in
// the substring "entity" by coincidence (Go identifiers rarely do this
// case-insensitively; the risk is asymmetric only for the lowercase form
// since PascalCase "Entity" as a suffix is not a common English-word
// collision).
func isEntityTypeName(name string) bool {
	return name == "entity" || strings.HasSuffix(name, "Entity")
}

func hasField(st *ast.StructType, name string) bool {
	if st.Fields == nil {
		return false
	}
	for _, f := range st.Fields.List {
		for _, n := range f.Names {
			if n.Name == name {
				return true
			}
		}
	}
	return false
}

// hasUniqueNaturalKey reports whether st declares a non-surrogate field
// carrying a `gorm:"...uniqueIndex..."` (or plain `unique`) tag — an
// actually-enforced database uniqueness constraint on a business-meaningful
// column, not a generated primary key. This is ONE of three preconditions
// Rule 1's control-plane allowlist exception requires (task-232 Task 19 fix
// round 1), not sufficient by itself: fix round 2 found that "declares SOME
// uniquely-constrained non-surrogate column" is a weaker claim than "the
// row IS the scoping dimension" — an idempotency-keyed audit/job record
// carries a real unique key too (its idempotency key) without being the
// top-level enumeration of anything. hasScopingDimensionMarker is the
// condition that closes that gap; see its own doc comment. A field
// literally named Id/ID is excluded even if tagged uniqueIndex, since that
// is the surrogate-PK convention every entity in the fleet already carries
// (`gorm:"type:uuid;default: uuid_generate_v4()"` etc.) and proves nothing
// about being a scoping dimension — see environments.Entity's Name field
// (task-19-brief.md Step 3) for the shape this is meant to recognize.
func hasUniqueNaturalKey(st *ast.StructType) bool {
	if st.Fields == nil {
		return false
	}
	for _, f := range st.Fields.List {
		if f.Tag == nil {
			continue
		}
		tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Get("gorm")
		if !strings.Contains(tag, "uniqueIndex") && !strings.Contains(tag, "unique") {
			continue
		}
		for _, n := range f.Names {
			if n.Name != "Id" && n.Name != "ID" {
				return true
			}
		}
	}
	return false
}

// scopingDimensionMarkerName is the method name an entity author must
// declare on the entity itself to claim "this row IS the scoping
// dimension" (task-232 Task 19 fix round 3). hasUniqueNaturalKey alone
// proved to be a proxy for that claim, not the claim: any control-plane row
// with an unrelated unique key (an idempotency-keyed audit/job record, an
// event log deduplicated by request id) satisfies "declares a uniquely-
// constrained non-surrogate column" without being the top-level
// enumeration at all — a fix-round-2 re-review built exactly that shape
// (RequestId `gorm:"uniqueIndex"` on an audit row) and it sailed through
// with no diagnostic. Struct SHAPE cannot reliably decide "is this the
// scoping dimension" — that is a semantic claim about what the row means,
// not a syntactic property of its columns, and each shape refinement is
// one clever entity away from the next bypass.
//
// So the exception now requires the entity author to STATE the claim, at
// the entity's own declaration site, in a form a reviewer reading that
// file will see and a `grep -r ScopingDimension` finds fleet-wide: a
// zero-value marker method, `func (Entity) ScopingDimension() {}`. This is
// requirement 3 of 3 (alongside the allowlist.txt entry and
// hasUniqueNaturalKey) — all three must hold for the exception to apply,
// so an accidental exemption needs three independent things to align by
// coincidence, and a deliberate one is a loud, reviewable act in the
// entity's own diff rather than prose in a text file far from the code.
const scopingDimensionMarkerName = "ScopingDimension"

// hasScopingDimensionMarker reports whether the package declares a method
// named ScopingDimension on the type entityName (value or pointer
// receiver), anywhere in the package's files. A method can only be
// declared in the same package as its receiver type in Go, so this is
// already scoped to "the entity's own package" without needing a same-file
// restriction — matching by the receiver's literal AST identifier, not
// through embedding promotion or a type alias, per fix round 3's
// independent-reviewer probes. NOT an enforced invariant: today's two real
// instances (environments.Entity, testEntity) happen to declare the method
// directly beside the struct in the same file, but a marker declared in a
// separate file of the same package is accepted by design (fix round 4
// confirmed this is intentional, not a gap) — this function only requires
// package-level co-location, so a reviewer diffing the entity's own file
// may not always see the marker in that same diff.
func hasScopingDimensionMarker(pass *analysis.Pass, entityName string) bool {
	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			if fd.Name.Name != scopingDimensionMarkerName {
				continue
			}
			if receiverTypeName(fd.Recv.List[0].Type) == entityName {
				return true
			}
		}
	}
	return false
}

// receiverTypeName extracts the bare type name from a method receiver's
// type expression, unwrapping a pointer receiver (`*Entity`) to its
// identifier the same way a value receiver (`Entity`) already is one.
func receiverTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// serviceFromPkgPath extracts the atlas-<name> service module from a Go
// import path. Every services/atlas-<name>/... module in the fleet is
// declared `module atlas-<name>` (bare, unqualified) — confirmed at
// services/atlas-{ban,quest,configurations,tenants}/.../go.mod — so the
// package path's own first segment IS the service name for every service
// module, and never for a libs/ or tools/ one (those are fully qualified
// under github.com/Chronicle20/atlas/...). Import path is stable across
// `go vet` invocation styles (relative-cwd vs. absolute file paths); a file
// path is not, which is why this keys off Pkg.Path() rather than a
// filesystem path — the same reason callsiteKey below does.
func serviceFromPkgPath(pkgPath string) (string, bool) {
	svc, _, _ := strings.Cut(pkgPath, "/")
	if !strings.HasPrefix(svc, "atlas-") {
		return "", false
	}
	return svc, true
}

// entityAllowlistKey is pkg.Path() + the entity.go base file name — stable
// regardless of the driver's working directory. See serviceFromPkgPath.
func entityAllowlistKey(pass *analysis.Pass, pos token.Pos) string {
	filename := pass.Fset.Position(pos).Filename
	base := filename
	if idx := strings.LastIndexAny(base, "/\\"); idx >= 0 {
		base = base[idx+1:]
	}
	return pass.Pkg.Path() + "/" + base
}

// --- Rule 2: call-site level ---

func checkCallSites(pass *analysis.Pass, insp *inspector.Inspector) {
	// Pass 1: mark every CallExpr that is itself the receiver of another
	// call further out in the same chain — those are never the "terminal"
	// call of a chain and are skipped, so one query chain (however many
	// verbs it contains) is reported exactly once, at its outermost call.
	chained := map[*ast.CallExpr]bool{}
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			chained[inner] = true
		}
	})

	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		checkWithoutTenantFilter(pass, call)
		if chained[call] {
			return
		}
		checkUnwrappedDbCall(pass, call)
	})
}

// checkWithoutTenantFilter flags every call site of the fleet-wide opt-out,
// outside libs/atlas-database itself. Purely syntactic (matches the audit's
// own "WithoutTenantFilter fleet grep", method 2 of §3) — a call site that
// explicitly opts out of the callback is unscoped by construction regardless
// of how the tenant-less context arrived.
func checkWithoutTenantFilter(pass *analysis.Pass, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if sel.Sel.Name != "WithoutTenantFilter" {
		return
	}
	pos := pass.Fset.Position(call.Pos())
	if isTestFile(pos) {
		// Test fixtures legitimately seed data across tenants with
		// WithoutTenantFilter (e.g. services/atlas-mts/.../task/
		// periodic_test.go:62) — not a production query path. The audit's
		// own raw-SQL sweep excludes _test.go for the same reason.
		return
	}
	key := callsiteKey(pass, pos)
	if reason, ok := CallsiteAllowlist[key]; ok {
		_ = reason
		return
	}
	pass.Reportf(call.Pos(), "call site opts out of tenant scoping (WithoutTenantFilter) with no allowlist entry")
}

// checkUnwrappedDbCall flags a terminal GORM-verb call chain rooted at a
// struct field literally named `db`/`DB` that never reaches a
// `.WithContext(` — the "silent-no-op" mechanism (tenant_scope.go:60,69-73).
// Deliberately field-name-syntactic rather than type-checked, matching the
// audit's own method: a `db *gorm.DB` struct field is exactly the shape the
// audit's struct-held db+ctx field scan targeted, and type-checking would
// require carrying a gorm stub package with no additional discriminating
// power over the field name.
func checkUnwrappedDbCall(pass *analysis.Pass, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !gormVerbs[sel.Sel.Name] {
		return
	}
	if chainHasWithContext(call) {
		return
	}
	if !chainRootIsDbField(call) {
		return
	}
	pos := pass.Fset.Position(call.Pos())
	if isTestFile(pos) {
		return
	}
	key := callsiteKey(pass, pos)
	if reason, ok := CallsiteAllowlist[key]; ok {
		_ = reason
		return
	}
	pass.Reportf(call.Pos(), "GORM call on a db field with no .WithContext in its chain — collapses to context.Background() and silently skips tenant scoping")
}

func chainHasWithContext(expr ast.Expr) bool {
	for {
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			return false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if sel.Sel.Name == "WithContext" {
			return true
		}
		expr = sel.X
	}
}

// chainRootIsDbField reports whether a call chain's root is a `db`/`DB`
// struct field selector (e.g. `s.db.Model(...)`).
//
// Known limitation (fix round 2, MINOR): this only recognizes a struct field
// literally named `db`/`DB`. A package-level `var db *gorm.DB` (an
// *ast.Ident root, not a *ast.SelectorExpr) or an accessor like `getDB()`
// (a *ast.CallExpr root whose Fun is not itself a chained GORM verb) both
// evade this check — the chain root never matches the SelectorExpr shape
// below. No such occurrence exists fleet-wide as of fix round 2 (every
// db-holding struct found uses a `db`/`DB` field per query-scope-audit.md),
// so this is a documented gap rather than a live defect. Widening to cover
// package-level vars or accessor calls is deferred: doing so without type
// information risks false positives on unrelated identifiers/functions
// named `db`/`getDB` that have nothing to do with GORM.
func chainRootIsDbField(expr ast.Expr) bool {
	for {
		switch e := expr.(type) {
		case *ast.CallExpr:
			sel, ok := e.Fun.(*ast.SelectorExpr)
			if !ok {
				return false
			}
			expr = sel.X
		case *ast.SelectorExpr:
			if e.Sel.Name == "db" || e.Sel.Name == "DB" {
				return true
			}
			expr = e.X
		default:
			return false
		}
	}
}

// callsiteKey identifies one call site stably across the module-relative or
// absolute paths different `go vet` invocations may report: the package's
// own import path plus the file's base name and line number. Package import
// path is stable regardless of the driver's working directory (it comes
// from module resolution, not the filesystem path reported for the
// position) — the same reason tools/rediskeyguard keys its allowlist off
// `pass.Pkg.Path()` rather than a file path.
// isTestFile reports whether pos falls in a _test.go file — excluded from
// both call-site rules, matching the audit's own methodology (its raw-SQL
// bypass grep explicitly excludes _test.go). Test fixtures legitimately
// seed cross-tenant data or call query helpers without a request-scoped
// context; that is not a production tenant-scope defect.
func isTestFile(pos token.Position) bool {
	return strings.HasSuffix(pos.Filename, "_test.go")
}

func callsiteKey(pass *analysis.Pass, pos token.Position) string {
	base := pos.Filename
	if idx := strings.LastIndexByte(base, '/'); idx >= 0 {
		base = base[idx+1:]
	}
	if idx := strings.LastIndexByte(base, '\\'); idx >= 0 {
		base = base[idx+1:]
	}
	return pass.Pkg.Path() + "/" + base + ":" + itoa(pos.Line)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
