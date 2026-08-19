package env

import (
	"testing"
	"time"
)

func active(name, baseline, ns string, overrides map[string]string) Record {
	return Record{
		Name: Id(name), Baseline: Id(baseline), Namespace: ns,
		Overrides: overrides, Phase: PhaseActive,
	}
}

func TestLegacyEnvironmentResolvesToTheLocalDeployment(t *testing.T) {
	// FR-1.8: with no records, every query returns the legacy answer.
	r := NewMapRegistry(Id("main"), time.Now)

	if !r.IsOwner(Id(""), "atlas-monsters") {
		t.Fatal("IsOwner(\"\") = false, want true (legacy owns everything)")
	}
	if !r.IsActive(Id("")) {
		t.Fatal("IsActive(\"\") = false, want true")
	}
	if got := r.EnvironmentsOwnedBy("atlas-monsters"); len(got) != 1 || got[0] != Id("") {
		t.Fatalf("EnvironmentsOwnedBy = %v, want [\"\"]", got)
	}
}

func TestRecordProvisionableAcrossPhases(t *testing.T) {
	cases := []struct {
		phase string
		want  bool
	}{
		{PhaseProvisioning, true},
		{PhaseActive, true},
		{PhaseDeactivating, false},
		{PhaseDeleted, false},
	}
	for _, c := range cases {
		rec := Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: c.phase}
		if got := rec.Provisionable(); got != c.want {
			t.Errorf("Provisionable() phase=%q = %v, want %v", c.phase, got, c.want)
		}
	}
}

func TestIsProvisionableAcrossPhasesAndUnknownAndLegacy(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(Record{Name: "pr-provisioning", Baseline: "main", Namespace: "atlas-pr-provisioning", Phase: PhaseProvisioning})
	r.Apply(active("pr-active", "main", "atlas-pr-active", nil))
	// DEACTIVATING and DELETED aren't directly Apply-able (DELETED removes the
	// record), so exercise them via Record.Provisionable() above, and confirm
	// the registry's ok-check path here with a DEACTIVATING record projected
	// as-is (Apply stores any phase other than DELETED verbatim).
	r.Apply(Record{Name: "pr-deactivating", Baseline: "main", Namespace: "atlas-pr-deactivating", Phase: PhaseDeactivating})

	if !r.IsProvisionable(Id("")) {
		t.Fatal("IsProvisionable(\"\") = false, want true (FR-1.8 legacy short-circuit)")
	}
	if !r.IsProvisionable(Id("pr-provisioning")) {
		t.Fatal("PROVISIONING environment reported not provisionable")
	}
	if !r.IsProvisionable(Id("pr-active")) {
		t.Fatal("ACTIVE environment reported not provisionable")
	}
	if r.IsProvisionable(Id("pr-deactivating")) {
		t.Fatal("DEACTIVATING environment reported provisionable")
	}
	if r.IsProvisionable(Id("pr-999")) {
		t.Fatal("unknown environment reported provisionable")
	}

	// DELETED: Apply with PhaseDeleted removes the record entirely, so the
	// lookup falls into the same not-found path as unknown.
	r.Apply(Record{Name: "pr-deleted", Baseline: "main", Namespace: "atlas-pr-deleted", Phase: PhaseDeleted})
	if r.IsProvisionable(Id("pr-deleted")) {
		t.Fatal("DELETED environment reported provisionable")
	}
}

func TestServiceNamespaceFallsBackToTheBaseline(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("main", "main", "atlas-main", nil))
	r.Apply(active("pr-123", "main", "atlas-pr-123", map[string]string{
		"atlas-character": "atlas-pr-123",
	}))

	got, err := r.ServiceNamespace(Id("pr-123"), "atlas-character")
	if err != nil || got != "atlas-pr-123" {
		t.Fatalf("override: got (%q, %v), want (\"atlas-pr-123\", nil)", got, err)
	}
	got, err = r.ServiceNamespace(Id("pr-123"), "atlas-monsters")
	if err != nil || got != "atlas-main" {
		t.Fatalf("fallback: got (%q, %v), want (\"atlas-main\", nil)", got, err)
	}
}

func TestEnvironmentNamespaceNeverFallsBackToTheBaseline(t *testing.T) {
	// The bug this test exists to prevent: an environment's OWN namespace is
	// where its OWN ingress lives. It is not subject to the per-service
	// override/baseline decision, because the record's `overrides` map does
	// not (and must not) list atlas-ingress. If this returned the baseline's
	// namespace, a baseline pod handling a pr-123 operation would send its
	// next REST call into main and the operation would silently change
	// environment mid-flight (G4).
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("main", "main", "atlas-main", nil))
	r.Apply(active("pr-123", "main", "atlas-pr-123", map[string]string{
		"atlas-character": "atlas-pr-123", // note: no atlas-ingress entry
	}))

	got, err := r.EnvironmentNamespace(Id("pr-123"))
	if err != nil || got != "atlas-pr-123" {
		t.Fatalf("got (%q, %v), want (\"atlas-pr-123\", nil)", got, err)
	}

	// And the contrast, stated explicitly so a future refactor that collapses
	// the two queries fails here rather than in production.
	svcNs, err := r.ServiceNamespace(Id("pr-123"), "atlas-ingress")
	if err != nil {
		t.Fatalf("ServiceNamespace: %v", err)
	}
	if svcNs == got {
		t.Fatal("ServiceNamespace(e, \"atlas-ingress\") happens to equal EnvironmentNamespace(e); " +
			"the fixture no longer distinguishes the two queries — fix the fixture, not the assertion")
	}
}

func TestNamespaceQueriesNeverHardCodeMain(t *testing.T) {
	// FR-1.5: a second baseline must require no code change.
	r := NewMapRegistry(Id("staging"), time.Now)
	r.Apply(active("staging", "staging", "atlas-staging", nil))
	r.Apply(active("pr-9", "staging", "atlas-pr-9", nil))

	got, err := r.ServiceNamespace(Id("pr-9"), "atlas-monsters")
	if err != nil || got != "atlas-staging" {
		t.Fatalf("ServiceNamespace: got (%q, %v), want (\"atlas-staging\", nil)", got, err)
	}
	got, err = r.EnvironmentNamespace(Id("pr-9"))
	if err != nil || got != "atlas-pr-9" {
		t.Fatalf("EnvironmentNamespace: got (%q, %v), want (\"atlas-pr-9\", nil)", got, err)
	}
}

func TestNamespaceQueriesOfAnUnknownEnvironmentError(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("main", "main", "atlas-main", nil))

	if _, err := r.ServiceNamespace(Id("pr-999"), "atlas-monsters"); err == nil {
		t.Fatal("ServiceNamespace resolved an unknown environment; want an error (D4 fail closed)")
	}
	if _, err := r.EnvironmentNamespace(Id("pr-999")); err == nil {
		t.Fatal("EnvironmentNamespace resolved an unknown environment; want an error (D4 fail closed)")
	}
}

func TestIsOwnerIsExactlyOneDeployment(t *testing.T) {
	// FR-4.6: for a given (environment, service) exactly one deployment
	// satisfies IsOwner, and every pod projects the same log.
	overrides := map[string]string{"atlas-character": "atlas-pr-123"}
	baselinePod := NewMapRegistry(Id("main"), time.Now)
	overridePod := NewMapRegistry(Id("pr-123"), time.Now)
	for _, r := range []*MapRegistry{baselinePod, overridePod} {
		r.Apply(active("main", "main", "atlas-main", nil))
		r.Apply(active("pr-123", "main", "atlas-pr-123", overrides))
	}

	if baselinePod.IsOwner(Id("pr-123"), "atlas-character") {
		t.Fatal("baseline claims ownership of an overridden service (FR-6.3)")
	}
	if !overridePod.IsOwner(Id("pr-123"), "atlas-character") {
		t.Fatal("override does not own its own service")
	}
	if !baselinePod.IsOwner(Id("pr-123"), "atlas-monsters") {
		t.Fatal("baseline does not own a non-overridden service for pr-123")
	}
	if overridePod.IsOwner(Id("pr-123"), "atlas-monsters") {
		t.Fatal("override claims a service it does not override")
	}
}

func TestNonActivePhaseIsNotActiveAndNotOwned(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	rec := active("pr-123", "main", "atlas-pr-123", nil)
	rec.Phase = PhaseProvisioning
	r.Apply(rec)

	if r.IsActive(Id("pr-123")) {
		t.Fatal("PROVISIONING environment reported ACTIVE")
	}
	if r.IsOwner(Id("pr-123"), "atlas-monsters") {
		t.Fatal("PROVISIONING environment reported owned; overrides must receive no work (FR-5.2)")
	}
}

func TestTombstoneRemovesTheEnvironment(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("pr-123", "main", "atlas-pr-123", nil))
	r.ApplyTombstone(Id("pr-123"))

	if r.IsActive(Id("pr-123")) {
		t.Fatal("tombstoned environment still ACTIVE (FR-5.7)")
	}
}

func TestEnvironmentsOwnedByExcludesOverriddenServices(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("main", "main", "atlas-main", nil))
	r.Apply(active("pr-1", "main", "atlas-pr-1", map[string]string{"atlas-character": "atlas-pr-1"}))
	r.Apply(active("pr-2", "main", "atlas-pr-2", nil))

	got := r.EnvironmentsOwnedBy("atlas-character")
	if len(got) != 2 { // main + pr-2, NOT pr-1
		t.Fatalf("EnvironmentsOwnedBy(atlas-character) = %v, want main and pr-2", got)
	}
}

func TestEnvironmentsOwnedByExcludesNonActiveEnvironments(t *testing.T) {
	// FR-5.2: autonomous iteration (Tasks 22-27) walks EnvironmentsOwnedBy to
	// decide what work to do. A non-active baseline-owned environment (its
	// namespace/ingress not yet ready, or being torn down) must never be
	// returned, or a service would act on it before it exists / after it is
	// gone.
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("main", "main", "atlas-main", nil))
	provisioning := active("pr-3", "main", "atlas-pr-3", nil)
	provisioning.Phase = PhaseProvisioning
	r.Apply(provisioning)

	got := r.EnvironmentsOwnedBy("atlas-character")
	if len(got) != 1 || got[0] != Id("main") {
		t.Fatalf("EnvironmentsOwnedBy(atlas-character) = %v, want only main; PROVISIONING pr-3 must be excluded", got)
	}
}

func TestStaleAfterFourMissedHeartbeats(t *testing.T) {
	now := time.Unix(0, 0)
	r := NewMapRegistry(Id("main"), func() time.Time { return now })
	r.Observe(now)

	if r.Stale() {
		t.Fatal("fresh registry reported stale")
	}
	now = now.Add(119 * time.Second)
	if r.Stale() {
		t.Fatal("stale at 119s; bound is 120s")
	}
	now = now.Add(1 * time.Second) // exactly 120s since Observe
	if r.Stale() {
		t.Fatal("stale at exactly 120s; design §4.3 is \"stale AFTER 120s\", so the boundary itself is not yet stale")
	}
	now = now.Add(1 * time.Second) // 121s
	if !r.Stale() {
		t.Fatal("not stale at 121s; bound is 120s")
	}
}

func TestCurrentRegistryIsNeverNil(t *testing.T) {
	if CurrentRegistry() == nil {
		t.Fatal("CurrentRegistry() = nil before SetRegistry; must be the legacy no-op")
	}
	if !CurrentRegistry().IsOwner(Id(""), "atlas-anything") {
		t.Fatal("the default registry is not the legacy no-op")
	}
}

func TestSetRegistryNilRestoresTheLegacyNoOp(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: PhaseActive})
	SetRegistry(r)
	t.Cleanup(func() { SetRegistry(nil) })

	if CurrentRegistry().IsActive(Id("pr-123")) == false {
		t.Fatal("registry was not installed")
	}

	SetRegistry(nil)

	if CurrentRegistry() == nil {
		t.Fatal("SetRegistry(nil) stored nil; must substitute the legacy no-op")
	}
	if !CurrentRegistry().IsActive(Id("pr-123")) {
		t.Fatal("the legacy no-op must answer IsActive true for any id (FR-1.8)")
	}
	if !CurrentRegistry().IsOwner(Id(""), "atlas-anything") {
		t.Fatal("SetRegistry(nil) did not restore the legacy no-op")
	}
}
