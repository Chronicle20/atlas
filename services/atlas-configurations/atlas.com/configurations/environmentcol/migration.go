// Package environmentcol adds the environment dimension to atlas-configurations'
// control-plane tables (task-232 D5). It is schema + backfill only: the
// scoped read/write paths land in a later task.
package environmentcol

import (
	"fmt"
	"os"

	"gorm.io/gorm"
)

// BackfillEnvironment assigns every control-plane row with an empty
// environment to the baseline. Idempotent: rows already carrying an
// environment are untouched, so re-running a migration is safe.
func BackfillEnvironment(db *gorm.DB, baseline string) error {
	for _, table := range []string{"tenants", "tenant_history", "templates", "services", "service_history"} {
		if err := db.Exec(
			"UPDATE "+table+" SET environment = ? WHERE environment = '' OR environment IS NULL",
			baseline,
		).Error; err != nil {
			return fmt.Errorf("backfill %s: %w", table, err)
		}
	}
	return nil
}

// Migration runs the backfill as part of the service's migration list. Must
// be registered after templates.Migration, tenants.Migration, and
// services.Migration in main.go — it backfills the columns those create.
func Migration(db *gorm.DB) error {
	return BackfillEnvironment(db, environmentBaseline())
}

// environmentBaseline is the environment existing rows belong to. main is
// the only baseline this repo ships; the literal appears here and nowhere
// else (FR-1.5 keeps the runtime baseline a record field).
func environmentBaseline() string {
	if v := os.Getenv("ATLAS_ENVIRONMENT"); v != "" {
		return v
	}
	return "main"
}
