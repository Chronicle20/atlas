package tenant

import (
	"os"

	"gorm.io/gorm"
)

// BackfillEnvironment assigns every tenant row with an empty environment to
// the baseline. Idempotent: rows already carrying an environment are
// untouched, so re-running the migration is safe.
func BackfillEnvironment(db *gorm.DB, baseline string) error {
	return db.Exec(
		"UPDATE tenants SET environment = ? WHERE environment = '' OR environment IS NULL",
		baseline,
	).Error
}

// EnvironmentMigration runs the backfill as part of the service's migration
// list. Must be registered after MigrateEntities in main.go — it backfills
// the column that creates.
func EnvironmentMigration(db *gorm.DB) error {
	return BackfillEnvironment(db, environmentBaseline())
}

// environmentBaseline is the environment existing rows belong to. main is
// the only baseline this repo ships.
func environmentBaseline() string {
	if v := os.Getenv("ATLAS_ENVIRONMENT"); v != "" {
		return v
	}
	return "main"
}
