// Package scope applies task-232's environment scoping to control-plane
// reads and writes (design §8.1). It implements the STRICT strategy used by
// tenants and services: a row belongs to exactly one environment, and every
// read filters to the caller's, every write is rejected unless the caller
// and the target agree.
//
// The templates table uses a DIFFERENT strategy - baseline fallback - which
// is templates-specific (the overlay key is a version, not an environment)
// and lives in the templates package's overlay.go, built on top of Strict.
package scope

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// ErrCrossEnvironmentWrite is returned by AuthorizeWrite when the caller's
// environment does not match the write's target environment. The REST layer
// maps it to 403 Forbidden.
var ErrCrossEnvironmentWrite = errors.New("write targets another environment")

// Strict filters db to rows owned by e. An empty e is the legacy value and
// applies no filter (FR-1.8) - unfiltered, byte-identical to pre-change
// behaviour.
func Strict(db *gorm.DB, e env.Id) *gorm.DB {
	if e == "" {
		return db
	}
	return db.Where("environment = ?", string(e))
}

// AuthorizeWrite returns ErrCrossEnvironmentWrite when target != caller,
// except that a legacy caller ("") is always authorized - symmetric with
// Strict, which applies no filter for a legacy caller's reads. A legacy
// deployment (no ENVIRONMENT header) must remain byte-identical to
// pre-change behaviour (FR-1.8), and the environment-backfill migration
// stamps every pre-existing row with the baseline environment (e.g.
// "main"), never "" - so restricting "" to only ever match "" would reject
// every legacy write against a backfilled row.
func AuthorizeWrite(caller env.Id, target env.Id) error {
	if caller == "" || caller == target {
		return nil
	}
	return fmt.Errorf("%w: caller=%q target=%q", ErrCrossEnvironmentWrite, caller, target)
}
