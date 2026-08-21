package env

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReconcileAgreesWhenHeaderMatchesTenant(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.ApplyTenant("t-1", Id("pr-123"))

	got, err := Reconcile(r, Id("pr-123"), "t-1")
	if err != nil || got != Id("pr-123") {
		t.Fatalf("got (%q, %v), want (\"pr-123\", nil)", got, err)
	}
}

func TestReconcileRejectsADisagreement(t *testing.T) {
	// FR-7.7: a mismatch is a hard error, not a reconciliation. Silently
	// preferring either side is how an operation changes environment
	// mid-flight.
	r := NewMapRegistry(Id("main"), time.Now)
	r.ApplyTenant("t-1", Id("pr-123"))

	if _, err := Reconcile(r, Id("main"), "t-1"); !errors.Is(err, ErrEnvironmentMismatch) {
		t.Fatalf("got %v, want ErrEnvironmentMismatch", err)
	}
}

func TestReconcileDerivesFromTenantWhenNoHeaderIsPresent(t *testing.T) {
	// The autonomous / persisted-work recovery path (design §8.3): a saga
	// sweeper reads a row, gets a tenant, and needs an environment.
	r := NewMapRegistry(Id("main"), time.Now)
	r.ApplyTenant("t-1", Id("pr-123"))

	got, err := Reconcile(r, Id(""), "t-1")
	if err != nil || got != Id("pr-123") {
		t.Fatalf("got (%q, %v), want (\"pr-123\", nil)", got, err)
	}
}

func TestReconcileWithAnUnknownTenantTrustsTheHeader(t *testing.T) {
	// A tenant the projection has not seen yet is possible during
	// activation (design §7.3): the tenant and environment records travel
	// on different topics. Trusting the header here does NOT weaken D4 —
	// the ownership gate still rejects an unknown ENVIRONMENT.
	r := NewMapRegistry(Id("main"), time.Now)

	got, err := Reconcile(r, Id("pr-123"), "t-unknown")
	if err != nil || got != Id("pr-123") {
		t.Fatalf("got (%q, %v), want (\"pr-123\", nil)", got, err)
	}
}

func TestReconcileWithNeitherIsTheLegacyValue(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	got, err := Reconcile(r, Id(""), "")
	if err != nil || got != Id("") {
		t.Fatalf("got (%q, %v), want (\"\", nil)", got, err)
	}
}

func TestReconcileTrustsTheHeaderForALegacyTenant(t *testing.T) {
	// FR-3.1: a tenant projected with an EMPTY environment is legacy, not a
	// hard mismatch against a sparse environment's header.
	r := NewMapRegistry(Id("main"), time.Now)
	r.ApplyTenant("t-1", Id(""))

	got, err := Reconcile(r, Id("pr-1411"), "t-1")
	if err != nil || got != Id("pr-1411") {
		t.Fatalf("got (%q, %v), want (\"pr-1411\", nil)", got, err)
	}
}

func TestReconcileStillRejectsTwoNonEmptyDisagreements(t *testing.T) {
	// FR-3.2: two distinct non-baseline environments are still a hard
	// mismatch; TestReconcileRejectsADisagreement covers header=main.
	r := NewMapRegistry(Id("main"), time.Now)
	r.ApplyTenant("t-1", Id("pr-123"))

	got, err := Reconcile(r, Id("pr-1411"), "t-1")
	if !errors.Is(err, ErrEnvironmentMismatch) || got != Id("") {
		t.Fatalf("got (%q, %v), want (\"\", ErrEnvironmentMismatch)", got, err)
	}
}

func TestReconcileWithALegacyTenantAndNoHeaderIsTheLegacyValue(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.ApplyTenant("t-1", Id(""))

	got, err := Reconcile(r, Id(""), "t-1")
	if err != nil || got != Id("") {
		t.Fatalf("got (%q, %v), want (\"\", nil)", got, err)
	}
}

func TestMismatchedIsFalseByDefault(t *testing.T) {
	if Mismatched(context.Background()) {
		t.Fatal("expected a fresh context to not be mismatched")
	}
}

func TestWithMismatchMarksTheContext(t *testing.T) {
	ctx := WithMismatch(context.Background())
	if !Mismatched(ctx) {
		t.Fatal("expected WithMismatch to make Mismatched report true")
	}
}

func TestMustFromContextStillReturnsTheHeaderIdAfterAMismatchIsRecorded(t *testing.T) {
	// EnvHeaderParser records a mismatch AND keeps the header's id on the
	// context (env.WithContext(ctx, id)) — WithMismatch only marks the
	// context, it never clears or overwrites the environment value. The
	// ownership gate depends on both being readable independently: the id
	// for labeling/logging, Mismatched for the drop decision.
	ctx := WithMismatch(WithContext(context.Background(), Id("pr-123")))
	if got := MustFromContext(ctx); got != Id("pr-123") {
		t.Fatalf("MustFromContext = %q, want %q", got, "pr-123")
	}
	if !Mismatched(ctx) {
		t.Fatal("expected Mismatched to still report true")
	}
}

// fakeRegistryWithoutTenantResolver implements Registry but deliberately
// not TenantResolver, exercising ForTenant's type-assertion fallback.
type fakeRegistryWithoutTenantResolver struct{}

func (fakeRegistryWithoutTenantResolver) EnvironmentNamespace(Id) (string, error) { return "", nil }
func (fakeRegistryWithoutTenantResolver) ServiceNamespace(Id, string) (string, error) {
	return "", nil
}
func (fakeRegistryWithoutTenantResolver) EnvironmentsOwnedBy(string) []Id { return nil }
func (fakeRegistryWithoutTenantResolver) IsOwner(Id, string) bool         { return false }
func (fakeRegistryWithoutTenantResolver) IsActive(Id) bool                { return false }
func (fakeRegistryWithoutTenantResolver) IsProvisionable(Id) bool         { return false }
func (fakeRegistryWithoutTenantResolver) Stale() bool                     { return false }
func (fakeRegistryWithoutTenantResolver) BaselineOf(Id) (Id, bool)        { return "", false }

func TestForTenant(t *testing.T) {
	withTenant := NewMapRegistry(Id("main"), time.Now)
	withTenant.ApplyTenant("t-1", Id("pr-1412"))

	legacyTenant := NewMapRegistry(Id("main"), time.Now)
	legacyTenant.ApplyTenant("t-1", Id(""))

	unknownTenant := NewMapRegistry(Id("main"), time.Now)

	tests := []struct {
		name     string
		r        Registry
		tenantId string
		self     Id
		want     Id
	}{
		{
			name:     "tenant belongs to another environment -> that environment, not self",
			r:        withTenant,
			tenantId: "t-1",
			self:     Id("main"),
			want:     Id("pr-1412"),
		},
		{
			name:     "empty tenantId -> self",
			r:        withTenant,
			tenantId: "",
			self:     Id("main"),
			want:     Id("main"),
		},
		{
			name:     "unknown tenant -> self",
			r:        unknownTenant,
			tenantId: "t-unknown",
			self:     Id("main"),
			want:     Id("main"),
		},
		{
			name:     "legacy tenant (empty projected environment) -> self",
			r:        legacyTenant,
			tenantId: "t-1",
			self:     Id("main"),
			want:     Id("main"),
		},
		{
			name:     "registry does not implement TenantResolver -> self",
			r:        fakeRegistryWithoutTenantResolver{},
			tenantId: "t-1",
			self:     Id("main"),
			want:     Id("main"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ForTenant(tt.r, tt.tenantId, tt.self); got != tt.want {
				t.Fatalf("ForTenant() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyTenantAndRemoveTenant(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)

	if _, ok := r.EnvironmentOfTenant("t-1"); ok {
		t.Fatalf("expected unknown tenant before ApplyTenant")
	}

	r.ApplyTenant("t-1", Id("pr-123"))
	got, ok := r.EnvironmentOfTenant("t-1")
	if !ok || got != Id("pr-123") {
		t.Fatalf("got (%q, %v), want (\"pr-123\", true)", got, ok)
	}

	r.RemoveTenant("t-1")
	if _, ok := r.EnvironmentOfTenant("t-1"); ok {
		t.Fatalf("expected unknown tenant after RemoveTenant")
	}
}
