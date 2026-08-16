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

// AuthorizeWrite returns ErrCrossEnvironmentWrite when target != caller.
// The legacy caller ("") only ever targets the legacy environment, so a
// legacy write is always authorized.
func AuthorizeWrite(caller env.Id, target env.Id) error {
	if caller == target {
		return nil
	}
	return fmt.Errorf("%w: caller=%q target=%q", ErrCrossEnvironmentWrite, caller, target)
}
