package configuration

import "testing"

// TestTaxWorkedExample pins the PRD's acceptance criterion: 10,000,000 staged
// delivers 9,600,000 at the 4% tier, and the 400,000 difference is destroyed —
// it is credited to nobody.
func TestTaxWorkedExample(t *testing.T) {
	tax, delivered := Tax(DefaultConfig(), 10_000_000)
	if tax != 400_000 {
		t.Errorf("tax: got %d, want 400000", tax)
	}
	if delivered != 9_600_000 {
		t.Errorf("delivered: got %d, want 9600000", delivered)
	}
	if tax+delivered != 10_000_000 {
		t.Errorf("tax+delivered: got %d, want 10000000", tax+delivered)
	}
}

// TestTaxTierBoundariesFromBothSides pins that each threshold is inclusive and
// the tier below it applies one meso lower.
func TestTaxTierBoundariesFromBothSides(t *testing.T) {
	cases := []struct {
		amount   uint32
		wantRate float64
	}{
		{100_000_000, 0.060},
		{99_999_999, 0.050},
		{25_000_000, 0.050},
		{24_999_999, 0.040},
		{10_000_000, 0.040},
		{9_999_999, 0.030},
		{5_000_000, 0.030},
		{4_999_999, 0.018},
		{1_000_000, 0.018},
		{999_999, 0.008},
		{100_000, 0.008},
		{99_999, 0.0},
		{0, 0.0},
	}
	for _, c := range cases {
		tax, delivered := Tax(DefaultConfig(), c.amount)
		want := uint32(float64(c.amount) * c.wantRate)
		if tax != want {
			t.Errorf("amount %d: tax got %d, want %d (rate %.3f)", c.amount, tax, want, c.wantRate)
		}
		if tax+delivered != c.amount {
			t.Errorf("amount %d: tax+delivered got %d, want %d", c.amount, tax+delivered, c.amount)
		}
	}
}

// TestTaxBelowLowestThresholdIsFree pins the 0% floor as its own case rather
// than leaving it implicit in the boundary table: every amount under the
// lowest tier (100,000) must be delivered whole.
func TestTaxBelowLowestThresholdIsFree(t *testing.T) {
	for _, amount := range []uint32{0, 1, 999, 50_000, 99_998, 99_999} {
		tax, delivered := Tax(DefaultConfig(), amount)
		if tax != 0 {
			t.Errorf("amount %d: tax got %d, want 0", amount, tax)
		}
		if delivered != amount {
			t.Errorf("amount %d: delivered got %d, want %d", amount, delivered, amount)
		}
	}
}

// TestTaxAtMaxUint32DoesNotOverflow pins that the widest possible staged
// amount is taxed without wrapping. float64 represents every uint32 exactly
// and rate <= 1 bounds the product below 2^32, so both the tax and the
// delivered remainder stay inside uint32.
func TestTaxAtMaxUint32DoesNotOverflow(t *testing.T) {
	const amount = uint32(4_294_967_295)
	tax, delivered := Tax(DefaultConfig(), amount)
	if tax != 257_698_037 {
		t.Errorf("tax: got %d, want 257698037", tax)
	}
	if delivered != 4_037_269_258 {
		t.Errorf("delivered: got %d, want 4037269258", delivered)
	}
	if tax+delivered != amount {
		t.Errorf("tax+delivered: got %d, want %d", tax+delivered, amount)
	}
}

// TestTaxNeverExceedsTheStagedAmount pins the safety property the settlement
// saga relies on: delivered is never negative (i.e. tax never wraps past the
// amount) for any tier in the default table.
func TestTaxNeverExceedsTheStagedAmount(t *testing.T) {
	amounts := []uint32{
		0, 1, 99_999, 100_000, 999_999, 1_000_000, 4_999_999, 5_000_000,
		9_999_999, 10_000_000, 24_999_999, 25_000_000, 99_999_999,
		100_000_000, 2_000_000_000, 4_294_967_295,
	}
	for _, amount := range amounts {
		tax, delivered := Tax(DefaultConfig(), amount)
		if tax > amount {
			t.Errorf("amount %d: tax %d exceeds the staged amount", amount, tax)
		}
		if delivered > amount {
			t.Errorf("amount %d: delivered %d exceeds the staged amount", amount, delivered)
		}
		if tax+delivered != amount {
			t.Errorf("amount %d: tax+delivered got %d", amount, tax+delivered)
		}
	}
}

// TestTaxDisabledDeductsNothing pins FR-9.1's master switch.
func TestTaxDisabledDeductsNothing(t *testing.T) {
	m := DefaultConfig().WithTaxEnabled(false)
	tax, delivered := Tax(m, 100_000_000)
	if tax != 0 {
		t.Errorf("tax: got %d, want 0", tax)
	}
	if delivered != 100_000_000 {
		t.Errorf("delivered: got %d, want 100000000", delivered)
	}
}

// TestValidateTiersAcceptsTheDefaultTable pins that the shipped table itself
// satisfies the rule the validator enforces — without this the fallback test
// below could pass against a table that is itself invalid.
func TestValidateTiersAcceptsTheDefaultTable(t *testing.T) {
	if err := ValidateTiers(DefaultConfig().TaxTiers()); err != nil {
		t.Fatalf("ValidateTiers rejected the shipped default table: %v", err)
	}
}

// TestInvalidTierTableFallsBackToDefaults pins FR-9.3 / design §6.5: a table
// whose thresholds are not strictly descending, or whose rate is outside [0,1],
// is rejected LOUDLY and the shipped defaults are used — never a silent accept.
func TestInvalidTierTableFallsBackToDefaults(t *testing.T) {
	cases := map[string][]Tier{
		"ascending thresholds": {{Threshold: 100_000, Rate: 0.008}, {Threshold: 1_000_000, Rate: 0.018}},
		"duplicate threshold":  {{Threshold: 100_000, Rate: 0.008}, {Threshold: 100_000, Rate: 0.018}},
		"rate above one":       {{Threshold: 100_000, Rate: 1.5}},
		"negative rate":        {{Threshold: 100_000, Rate: -0.1}},
	}
	for name, tiers := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ValidateTiers(tiers); err == nil {
				t.Fatal("ValidateTiers accepted an invalid table")
			}
			m := DefaultConfig().WithTaxTiers(tiers) // applies the fallback
			if len(m.TaxTiers()) != len(DefaultConfig().TaxTiers()) {
				t.Errorf("tier count: got %d, want the default table's %d", len(m.TaxTiers()), len(DefaultConfig().TaxTiers()))
			}
		})
	}
}

// TestValidTierTableIsAccepted pins the other half of WithTaxTiers: a
// well-formed table must actually be adopted, not quietly replaced by the
// defaults. Without this, WithTaxTiers falling back unconditionally would
// still pass the fallback test above.
func TestValidTierTableIsAccepted(t *testing.T) {
	tiers := []Tier{{Threshold: 1_000_000, Rate: 0.5}, {Threshold: 1_000, Rate: 0.1}}
	m := DefaultConfig().WithTaxTiers(tiers)
	if len(m.TaxTiers()) != 2 {
		t.Fatalf("tier count: got %d, want 2", len(m.TaxTiers()))
	}
	tax, delivered := Tax(m, 1_000_000)
	if tax != 500_000 || delivered != 500_000 {
		t.Errorf("Tax(1000000): got tax %d delivered %d, want 500000/500000", tax, delivered)
	}
	tax, delivered = Tax(m, 1_000)
	if tax != 100 || delivered != 900 {
		t.Errorf("Tax(1000): got tax %d delivered %d, want 100/900", tax, delivered)
	}
	tax, delivered = Tax(m, 999)
	if tax != 0 || delivered != 999 {
		t.Errorf("Tax(999): got tax %d delivered %d, want 0/999", tax, delivered)
	}
}

// TestTaxTiersGetterReturnsACopy pins the immutability contract: a caller that
// mutates the returned slice must not corrupt the Model's table.
func TestTaxTiersGetterReturnsACopy(t *testing.T) {
	m := DefaultConfig()
	got := m.TaxTiers()
	got[0] = Tier{Threshold: 1, Rate: 0.99}

	after := m.TaxTiers()
	if after[0].Threshold != 100_000_000 || after[0].Rate != 0.060 {
		t.Fatalf("Model tier table was mutated through the getter: %+v", after[0])
	}
}

// TestEmptyTierTableFallsBackToDefaults pins the defence in depth the review
// asked about: an EMPTY tax table must be treated like a malformed one and
// replaced by the shipped table, never read as "no tax". Without the
// len(tiers) == 0 arm in WithTaxTiers, an empty table would reach Tax(), whose
// loop would match nothing and return 0 for every amount — silently disabling
// meso tax collection tenant-wide.
func TestEmptyTierTableFallsBackToDefaults(t *testing.T) {
	cases := map[string]Model{
		"WithTaxTiers(nil)":           DefaultConfig().WithTaxTiers(nil),
		"WithTaxTiers(empty)":         DefaultConfig().WithTaxTiers([]Tier{}),
		"Extract with no wire tiers":  Extract(RestModel{TaxEnabled: true}),
		"Extract with an empty array": Extract(RestModel{TaxEnabled: true, TaxTiers: []TierRestModel{}}),
	}

	for name, m := range cases {
		t.Run(name, func(t *testing.T) {
			if len(m.TaxTiers()) != len(DefaultConfig().TaxTiers()) {
				t.Fatalf("tier count: got %d, want the default table's %d", len(m.TaxTiers()), len(DefaultConfig().TaxTiers()))
			}
			tax, delivered := Tax(m, 100_000_000)
			if tax != 6_000_000 || delivered != 94_000_000 {
				t.Errorf("Tax(100000000): got tax %d delivered %d, want 6000000/94000000 — an empty table silently disabled the tax", tax, delivered)
			}
		})
	}
}
