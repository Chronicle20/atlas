package seeder

import (
	"context"

	"gorm.io/gorm"
)

// Group declares one (POST /<prefix>/seed, GET /<prefix>/seed/status) pair.
type Group struct {
	Name       string // stored as seed_state.group_name; e.g. "drops"
	URLPrefix  string // e.g. "/drops" → routes POST /drops/seed
	Subdomains []SubdomainAny
	// AfterSeed, when non-nil, runs exactly once after a successful
	// Seed with the tenant-bearing seed context. Use it to emit a
	// domain event announcing that the group's data changed. Errors
	// are logged, not returned to the HTTP caller — the seed has
	// already committed by the time this runs.
	AfterSeed func(ctx context.Context, db *gorm.DB, res Result) error
}
