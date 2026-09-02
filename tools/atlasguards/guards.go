// Package atlasguards bundles the repo's go/analysis guards into a single
// vettool so CI type-checks the tree once instead of once per guard.
//
// Each guard used to get its own CI job, its own runner, and its own cold
// `go vet` sweep of the same 64 service modules. Type-checking — not analysis —
// is what those jobs spent their time on, and four jobs did it four times over
// an identical dependency graph. Registering every analyzer with one
// unitchecker binary makes the marginal cost of guards 2..N essentially zero:
// the packages are parsed and type-checked once, and each analyzer walks the
// same syntax trees.
//
// The scope split below is NOT an optimization — it preserves exactly what each
// guard used to analyze:
//
//	rediskeyguard      services/ only
//	outboxguard        services/ only
//	goroutineguard     services/ + libs/
//	buffdurationguard  services/ + libs/
//	scopeguard         services/ + libs/
//	topicguard         services/ + libs/
//
// Widening rediskeyguard or outboxguard to libs/ as a side effect of merging
// the jobs would be a behavior change smuggled in under a performance change,
// so the two sets stay distinct and get two binaries. Diagnostics stay
// attributable because every analyzer already prefixes its message with its own
// name ("rediskeyguard: ...", "outboxguard: ..." and so on).
//
// scopeguard joined services/ + libs/ in fix round 2 (task-232): its own
// tools/scope-guard.sh entry point has always swept both roots (Rule 2, the
// call-site check, is fleet-wide by design — libs/atlas-database itself has
// a live WithoutTenantFilter call site the rule must see), and enrolling it
// here is what makes CI's go-analyzer-guards job actually run it — before
// this it had no CI entry point at all, only tools/verify.sh's own
// change-gated call to tools/scope-guard.sh.
//
// topicguard (task-276) is services/ + libs/ from the start: its raw-env-
// topic-read diagnostic exists because libs/atlas-service's envregistry.go
// itself holds live os.Getenv(string(...)) call sites, and its
// token-not-in-manifest diagnostic must see every topic.Token constant the
// fleet declares, not just the services/ subset — a services-only
// registration would miss both.
package atlasguards

import (
	"golang.org/x/tools/go/analysis"

	"github.com/Chronicle20/atlas/tools/buffdurationguard"
	"github.com/Chronicle20/atlas/tools/goroutineguard"
	"github.com/Chronicle20/atlas/tools/outboxguard"
	"github.com/Chronicle20/atlas/tools/rediskeyguard"
	"github.com/Chronicle20/atlas/tools/scopeguard"
	"github.com/Chronicle20/atlas/tools/topicguard"
)

// Services is the analyzer set that applies to modules under services/.
func Services() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		rediskeyguard.Analyzer,
		outboxguard.Analyzer,
		goroutineguard.Analyzer,
		buffdurationguard.Analyzer,
		scopeguard.Analyzer,
		topicguard.Analyzer,
	}
}

// Libraries is the analyzer set that applies to modules under libs/.
func Libraries() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		goroutineguard.Analyzer,
		buffdurationguard.Analyzer,
		scopeguard.Analyzer,
		topicguard.Analyzer,
	}
}
