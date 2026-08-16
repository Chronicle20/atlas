// Package envguard implements the env-domain-guard analyzer (task-232
// NG5/FR-4.5): it bans importing github.com/Chronicle20/atlas/libs/atlas-env
// from a domain package. atlas-env carries process-wide environment routing
// state; letting a domain package import it directly would let environment
// selection leak into business logic instead of staying confined to the
// transport seams (main.go's bootstrap wiring, and the kafka/ and rest/
// packages that already carry request/message-scoped context).
//
// Allowed import sites:
//   - main.go (any package) — bootstrap wiring passes the registry down.
//   - any file under a kafka/ or rest/ directory — the transport layer,
//     where FR-4.5's REST/Kafka environment propagation (tasks 23-27) lives.
//   - a package on domainAllowlist below — two distinct, narrow shapes:
//     control-plane packages that OWN environment/tenant data as their
//     domain model (atlas-configurations, atlas-tenants), and the single
//     env.Self()-only exception design §4.3 carves out for a pod's own,
//     never-stale environment (atlas-data/runtime/ingest) — see the entry
//     itself for why that second shape is NOT "ownership" and must not be
//     used as precedent for a different env.Self() call site.
//
// Everything else under services/ that imports atlas-env is a domain
// package reaching for global environment state directly, which is exactly
// what NG5 rules out.
//
// Wired into the fleet via tools/env-domain-guard.sh (SCAN_ROOTS=services,
// via tools/lib/analyzer-guard.sh — the same shared `go vet -vettool=`
// driver rediskeyguard uses). cmd/envguard is the standalone singlechecker
// escape hatch; cmd/envguardvet is the vettool entry point the shared
// driver actually runs.
package envguard

import (
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// domainPkgPath is the import path this guard bans outside the allowed
// sites above.
const domainPkgPath = "github.com/Chronicle20/atlas/libs/atlas-env"

// domainAllowlist names packages permitted to import atlas-env directly
// outside main.go/kafka/rest, keyed by import path, with a written reason.
// Add an entry here — never merely in a report or commit message — for any
// future exception; this map is the only place the guard checks. Verify
// against source before adding: an allowlist entry is permanent and
// invisible, and this guard is the only thing that will ever re-examine it.
//
// Seven of the eight entries are a control-plane service that OWNS
// environment or tenant data as its domain model (design.md §8.1's STRICT
// scoping and FR-7.3's tenant→environment derivation) — not an ordinary
// service reaching for global routing state, which is exactly what NG5
// bans. The eighth (atlas-data/runtime/ingest) is a materially different
// shape: it does NOT own environment data, it calls env.Self() directly —
// see that entry for the distinct justification. Discovered at task-232/28:
// these imports already existed on this branch from tasks 18-20 and
// earlier, predating this guard.
var domainAllowlist = map[string]string{
	"atlas-configurations/scope": "implements design §8.1's STRICT " +
		"environment-scoping strategy (env.Id filter/authorize) shared by " +
		"the tenants/services/templates domain packages below; confirmed " +
		"scope.go:18.",
	"atlas-configurations/environments": "IS the environment CRUD/outbox " +
		"domain (Task 18/19) — builds and publishes env.Record itself; " +
		"confirmed processor.go:16, heartbeat.go:9.",
	"atlas-configurations/services": "control-plane services table scoped " +
		"by env.Id per design §8.1/V4; confirmed administrator.go:13, " +
		"provider.go:12.",
	"atlas-configurations/templates": "control-plane templates table, " +
		"baseline-fallback scoping per design V3; confirmed " +
		"administrator.go:12, overlay.go:11, processor.go:13, provider.go:11.",
	"atlas-configurations/tenants": "control-plane tenants table scoped by " +
		"env.Id per design §8.1; confirmed administrator.go:13, " +
		"processor.go:18, provider.go:12.",
	"atlas-data/runtime/ingest": "NOT an ownership case like the other " +
		"seven — heartbeat.go:43 and progress.go:63,116 call env.Self() " +
		"directly, reaching for the pod's own environment rather than " +
		"receiving one as data. Sanctioned specifically by design §4.3's " +
		"staleness carve-out (\"Environment == env.Self() -> Proceed. A " +
		"pod's own environment comes from an env var and cannot go " +
		"stale.\"), the same rule already documented at " +
		"ingestrun.go:114-118 for this package's Redis registries. Not " +
		"precedent for any other env.Self() call site — each needs its " +
		"own §4.3 justification.",
	"atlas-tenants/scope": "implements the same design §8.1 STRICT " +
		"environment-scoping strategy as atlas-configurations/scope, for " +
		"the tenant table itself; confirmed scope.go:18.",
	"atlas-tenants/tenant": "IS the tenant domain that carries the " +
		"per-tenant \"environment\" attribute FR-7.3 derives from (task-232 " +
		"R21-1); confirmed administrator.go:13, processor.go:13, " +
		"provider.go:12.",
}

// Analyzer is the env-domain-guard check, run via `go vet -vettool=` by
// tools/lib/analyzer-guard.sh (env-domain-guard.sh) and directly in tests
// via analysistest.
var Analyzer = &analysis.Analyzer{
	Name: "envdomainguard",
	Doc:  "bans importing libs/atlas-env from a domain package (task-232 NG5/FR-4.5): only main.go, files under kafka/ or rest/, and domainAllowlist entries may import it",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil || path != domainPkgPath {
				continue
			}
			filename := pass.Fset.Position(imp.Pos()).Filename
			if allowedImportSite(filename) {
				continue
			}
			// An external test package (package foo_test) type-checks as
			// its own package whose path carries a "_test" suffix — strip
			// it for the lookup so a domainAllowlist entry covers both the
			// package and its external test, without needing a second,
			// easy-to-miss "_test"-suffixed entry per package.
			pkgPath := strings.TrimSuffix(pass.Pkg.Path(), "_test")
			if reason, allowed := domainAllowlist[pkgPath]; allowed {
				_ = reason
				continue
			}
			pass.Reportf(imp.Pos(),
				"envdomainguard: atlas-env imported from a domain package; only main.go, files under kafka/ or rest/, or an entry in domainAllowlist (with a written reason) may import it (task-232 NG5/FR-4.5)")
		}
	}
	return nil, nil
}

// allowedImportSite reports whether filename is one of the two sites
// atlas-env may be imported from: main.go anywhere, or any file with a
// kafka/ or rest/ path segment.
func allowedImportSite(filename string) bool {
	if filepath.Base(filename) == "main.go" {
		return true
	}
	dir := filepath.ToSlash(filepath.Dir(filename))
	for _, segment := range strings.Split(dir, "/") {
		if segment == "kafka" || segment == "rest" {
			return true
		}
	}
	return false
}
