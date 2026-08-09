package configuration

import (
	"testing"
	"time"
)

// TestDefaultConfigKnobs pins the shipped fallback values from design §8. These
// are the values a tenant with no trade-configs resource runs on (FR-9.2), so a
// silent drift here changes live behaviour for every unconfigured tenant.
func TestDefaultConfigKnobs(t *testing.T) {
	m := DefaultConfig()

	if !m.TaxEnabled() {
		t.Error("TaxEnabled: got false, want true")
	}
	if m.MaxStagedItems() != 9 {
		t.Errorf("MaxStagedItems: got %d, want 9", m.MaxStagedItems())
	}
	if m.MinTradeLevel() != 0 {
		t.Errorf("MinTradeLevel: got %d, want 0", m.MinTradeLevel())
	}
	if m.ReservationTtl() != 300*time.Second {
		t.Errorf("ReservationTtl: got %s, want 5m0s", m.ReservationTtl())
	}
	if m.AttestationTimeout() != 5*time.Second {
		t.Errorf("AttestationTimeout: got %s, want 5s", m.AttestationTimeout())
	}
}

// TestDefaultConfigTaxTable pins the six default tax bands (design §8) exactly.
// A wrong boundary or rate here silently destroys player meso.
func TestDefaultConfigTaxTable(t *testing.T) {
	want := []Tier{
		{Threshold: 100_000_000, Rate: 0.060},
		{Threshold: 25_000_000, Rate: 0.050},
		{Threshold: 10_000_000, Rate: 0.040},
		{Threshold: 5_000_000, Rate: 0.030},
		{Threshold: 1_000_000, Rate: 0.018},
		{Threshold: 100_000, Rate: 0.008},
	}

	got := DefaultConfig().TaxTiers()
	if len(got) != len(want) {
		t.Fatalf("tier count: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tier %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestWithTransformsDoNotMutateTheReceiver pins the immutable-model contract:
// each With* returns a new Model and leaves the original untouched.
func TestWithTransformsDoNotMutateTheReceiver(t *testing.T) {
	base := DefaultConfig()

	got := base.
		WithTaxEnabled(false).
		WithMaxStagedItems(3).
		WithMinTradeLevel(20).
		WithReservationTtl(90 * time.Second).
		WithAttestationTimeout(11 * time.Second)

	if got.TaxEnabled() {
		t.Error("WithTaxEnabled(false): got true")
	}
	if got.MaxStagedItems() != 3 {
		t.Errorf("WithMaxStagedItems(3): got %d", got.MaxStagedItems())
	}
	if got.MinTradeLevel() != 20 {
		t.Errorf("WithMinTradeLevel(20): got %d", got.MinTradeLevel())
	}
	if got.ReservationTtl() != 90*time.Second {
		t.Errorf("WithReservationTtl(90s): got %s", got.ReservationTtl())
	}
	if got.AttestationTimeout() != 11*time.Second {
		t.Errorf("WithAttestationTimeout(11s): got %s", got.AttestationTimeout())
	}

	if !base.TaxEnabled() || base.MaxStagedItems() != 9 || base.MinTradeLevel() != 0 ||
		base.ReservationTtl() != 300*time.Second || base.AttestationTimeout() != 5*time.Second {
		t.Errorf("receiver was mutated by the With* chain: %+v", base)
	}
}

// TestExtractFoldsAbsentKnobsToDefaults pins that a partially-populated
// atlas-tenants payload never yields a nonsensical zero knob (a zero
// reservation TTL would expire every reservation instantly).
func TestExtractFoldsAbsentKnobsToDefaults(t *testing.T) {
	m := Extract(RestModel{TaxEnabled: true})

	d := DefaultConfig()
	if m.MaxStagedItems() != d.MaxStagedItems() {
		t.Errorf("MaxStagedItems: got %d, want %d", m.MaxStagedItems(), d.MaxStagedItems())
	}
	if m.ReservationTtl() != d.ReservationTtl() {
		t.Errorf("ReservationTtl: got %s, want %s", m.ReservationTtl(), d.ReservationTtl())
	}
	if m.AttestationTimeout() != d.AttestationTimeout() {
		t.Errorf("AttestationTimeout: got %s, want %s", m.AttestationTimeout(), d.AttestationTimeout())
	}
	if len(m.TaxTiers()) != len(d.TaxTiers()) {
		t.Errorf("TaxTiers: got %d tiers, want the default table's %d", len(m.TaxTiers()), len(d.TaxTiers()))
	}
	// minTradeLevel's default IS zero, so an absent value is indistinguishable
	// from an explicit 0 — both mean "no level restriction".
	if m.MinTradeLevel() != 0 {
		t.Errorf("MinTradeLevel: got %d, want 0", m.MinTradeLevel())
	}
}

// TestExtractHonoursAnExplicitlyDisabledTax pins that taxEnabled is NOT
// zero-folded: atlas-tenants always serialises the flag, so `false` on the
// wire means the operator disabled the tax and must be obeyed.
func TestExtractHonoursAnExplicitlyDisabledTax(t *testing.T) {
	m := Extract(RestModel{TaxEnabled: false})
	if m.TaxEnabled() {
		t.Fatal("Extract folded an explicit taxEnabled=false back to the default true")
	}
}

// TestExtractAdoptsAConfiguredTierTable pins that a valid tenant-supplied
// table replaces the defaults wholesale.
func TestExtractAdoptsAConfiguredTierTable(t *testing.T) {
	m := Extract(RestModel{
		TaxEnabled: true,
		TaxTiers: []TierRestModel{
			{Threshold: 2_000_000, Rate: 0.25},
			{Threshold: 1_000, Rate: 0.01},
		},
	})

	if len(m.TaxTiers()) != 2 {
		t.Fatalf("tier count: got %d, want 2", len(m.TaxTiers()))
	}
	tax, delivered := Tax(m, 2_000_000)
	if tax != 500_000 || delivered != 1_500_000 {
		t.Errorf("Tax(2000000): got tax %d delivered %d, want 500000/1500000", tax, delivered)
	}
}

// TestExtractRejectsAnInvalidTierTable pins FR-9.3 at the transport boundary:
// a tenant that configures a broken table falls back to the shipped defaults
// rather than running on it.
func TestExtractRejectsAnInvalidTierTable(t *testing.T) {
	m := Extract(RestModel{
		TaxEnabled: true,
		TaxTiers: []TierRestModel{
			{Threshold: 1_000, Rate: 0.01},
			{Threshold: 2_000_000, Rate: 0.25}, // ascending — invalid
		},
	})

	if len(m.TaxTiers()) != len(DefaultConfig().TaxTiers()) {
		t.Fatalf("tier count: got %d, want the default table's %d", len(m.TaxTiers()), len(DefaultConfig().TaxTiers()))
	}
}
