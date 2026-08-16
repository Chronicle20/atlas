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

// AuthorizeWrite returns ErrCrossEnvironmentWrite when target != caller.
// The legacy caller ("") only ever targets the legacy environment, so a
// legacy write is always authorized.
func AuthorizeWrite(caller env.Id, target env.Id) error {
	if caller == target {
		return nil
	}
	return fmt.Errorf("%w: caller=%q target=%q", ErrCrossEnvironmentWrite, caller, target)
}
