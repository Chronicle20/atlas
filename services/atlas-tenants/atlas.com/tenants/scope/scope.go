// Package scope applies task-232's environment scoping to atlas-tenants'
// reads and writes (design §8.1). It implements the STRICT strategy: a row
// belongs to exactly one environment, every read filters to the caller's,
// and every write is rejected unless the caller and the target agree.
//
// This is a deliberate copy of atlas-configurations/scope's package (Task
// 13), not an import of it - per CLAUDE.md, one service must not import
// another's package. Three ten-line functions is the right amount of
// duplication.
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
