// Package configuration holds the per-tenant trade configuration that
// atlas-trades reads from atlas-tenants' `trade-configs` resource, plus the
// meso-tax calculator that consumes it.
//
// Model is immutable: private fields, getters, and a With* transform per knob.
// A Model's tax-tier table is ALWAYS valid — the only two ways to install one
// (WithTaxTiers and, through it, Extract) run it through
// ValidateTiers and fall back to the shipped defaults on rejection. Tax()
// therefore does not need to defend against a rate outside [0, 1].
package configuration

import "time"

// Tier is one meso-tax band. Rate applies to amounts >= Threshold, with the
// FIRST matching (highest) tier winning — so the table must be strictly
// descending by Threshold.
type Tier struct {
	Threshold uint32  `json:"threshold"`
	Rate      float64 `json:"rate"`
}

// Model is the immutable per-tenant trade configuration (design §8).
type Model struct {
	taxEnabled         bool          // master switch for the meso tax (FR-9.1)
	taxTiers           []Tier        // strictly descending by Threshold; first match wins
	maxStagedItems     int           // per-side cap on staged item slots
	minTradeLevel      int           // minimum character level allowed to trade; 0 = unrestricted
	reservationTtl     time.Duration // how long an escrow reservation survives before it is released
	attestationTimeout time.Duration // deadline for both sides to attest before the trade aborts
}

func (m Model) TaxEnabled() bool {
	return m.taxEnabled
}

// TaxTiers returns a copy of the tax table so a caller cannot mutate the
// Model's own slice.
func (m Model) TaxTiers() []Tier {
	out := make([]Tier, len(m.taxTiers))
	copy(out, m.taxTiers)
	return out
}

func (m Model) MaxStagedItems() int {
	return m.maxStagedItems
}

func (m Model) MinTradeLevel() int {
	return m.minTradeLevel
}

func (m Model) ReservationTtl() time.Duration {
	return m.reservationTtl
}

func (m Model) AttestationTimeout() time.Duration {
	return m.attestationTimeout
}

// WithTaxEnabled returns a copy with the master tax switch set.
func (m Model) WithTaxEnabled(enabled bool) Model {
	m.taxEnabled = enabled
	return m
}

// WithTaxTiers returns a copy carrying the given tax table, or — when that
// table fails ValidateTiers — carrying the shipped default table instead
// (FR-9.3). Callers that need to know a table was rejected call ValidateTiers
// themselves and log; this transform only applies the fallback.
func (m Model) WithTaxTiers(tiers []Tier) Model {
	if ValidateTiers(tiers) != nil || len(tiers) == 0 {
		m.taxTiers = defaultTiers()
		return m
	}
	out := make([]Tier, len(tiers))
	copy(out, tiers)
	m.taxTiers = out
	return m
}

// WithMaxStagedItems returns a copy with the staged-item cap set.
func (m Model) WithMaxStagedItems(items int) Model {
	m.maxStagedItems = items
	return m
}

// WithMinTradeLevel returns a copy with the minimum trading level set.
func (m Model) WithMinTradeLevel(level int) Model {
	m.minTradeLevel = level
	return m
}

// WithReservationTtl returns a copy with the escrow reservation TTL set.
func (m Model) WithReservationTtl(ttl time.Duration) Model {
	m.reservationTtl = ttl
	return m
}

// WithAttestationTimeout returns a copy with the attestation deadline set.
func (m Model) WithAttestationTimeout(timeout time.Duration) Model {
	m.attestationTimeout = timeout
	return m
}

// defaultTiers returns a fresh copy of the shipped tax table (design §8). The
// bands are strictly descending and the first match wins, so an amount of
// exactly 100,000,000 pays 6% while 99,999,999 pays 5%, and anything below
// 100,000 pays nothing.
func defaultTiers() []Tier {
	return []Tier{
		{Threshold: 100_000_000, Rate: 0.060},
		{Threshold: 25_000_000, Rate: 0.050},
		{Threshold: 10_000_000, Rate: 0.040},
		{Threshold: 5_000_000, Rate: 0.030},
		{Threshold: 1_000_000, Rate: 0.018},
		{Threshold: 100_000, Rate: 0.008},
	}
}

// DefaultConfig is the shipped fallback used whenever a tenant has no
// trade-configs resource or its table fails validation. The service never
// hard-fails on a missing configuration resource (FR-9.2) and never silently
// disables trading.
func DefaultConfig() Model {
	return Model{
		taxEnabled:         true,
		taxTiers:           defaultTiers(),
		maxStagedItems:     9,
		minTradeLevel:      0,
		reservationTtl:     300 * time.Second,
		attestationTimeout: 5 * time.Second,
	}
}
