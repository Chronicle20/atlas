// Package configuration holds the per-tenant configuration atlas-character
// reads from atlas-tenants' `imprint-configs` resource (FR-2.6): how long a
// pending cash name-change / world-transfer request survives before the
// sweep expires and refunds it.
//
// This is atlas-character's first consumed tenant configuration, modelled
// file-for-file on services/atlas-trades/atlas.com/trades/configuration/ —
// the smallest complete instance of the pattern, since there is exactly one
// knob rather than trades' tax-tier table.
//
// Model is immutable: private fields, getters, and a With* transform.
package configuration

import "time"

// DefaultPendingExpiry is the shipped fallback lifetime (design §5.5 / FR-2.6):
// 7 days. It is used whenever a tenant has no imprint-configs resource, so an
// unseeded tenant keeps working rather than expiring every pending change
// instantly or failing outright.
const DefaultPendingExpiry = 168 * time.Hour

// Model is the immutable per-tenant imprint (pending-change) configuration.
type Model struct {
	pendingExpiry time.Duration
}

// PendingExpiry returns how long a pending request survives before the sweep
// expires and refunds it.
func (m Model) PendingExpiry() time.Duration {
	return m.pendingExpiry
}

// WithPendingExpiry returns a copy with the pending-change expiry set. A
// non-positive duration is rejected in favor of DefaultPendingExpiry — a 0h
// (or negative) expiry would expire every pending change immediately, which
// is the one failure mode this package must never produce.
func (m Model) WithPendingExpiry(d time.Duration) Model {
	if d <= 0 {
		m.pendingExpiry = DefaultPendingExpiry
		return m
	}
	m.pendingExpiry = d
	return m
}

// DefaultConfig is the shipped fallback used whenever a tenant has no
// imprint-configs resource. The service never hard-fails on a missing
// configuration resource — resolving configuration must never be the thing
// that blocks a pending-change request.
func DefaultConfig() Model {
	return Model{
		pendingExpiry: DefaultPendingExpiry,
	}
}
