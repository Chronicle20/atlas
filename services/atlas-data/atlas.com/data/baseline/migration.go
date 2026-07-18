package baseline

import "gorm.io/gorm"

// Restore lifecycle states persisted in tenant_baselines.status. A row is
// written as StatusRestoring BEFORE the first table COPY and flipped to
// StatusComplete only after every table + ANALYZE + finalize succeeds. An
// interrupted restore therefore leaves a durable StatusRestoring marker that
// startup reconciliation (Reconcile) re-runs — the fix for the atlas-pr-933
// half-restore, where a cancelled restore left item_string_search_index empty
// with no way to self-heal.
const (
	StatusRestoring = "restoring"
	StatusComplete  = "complete"
)

type tenantBaseline struct {
	TenantID       string `gorm:"primaryKey;type:uuid;column:tenant_id"`
	Region         string `gorm:"not null;column:region"`
	MajorVersion   int    `gorm:"not null;column:major_version"`
	MinorVersion   int    `gorm:"not null;column:minor_version"`
	BaselineSha256 string `gorm:"not null;column:baseline_sha256"`
	RestoredAt     string `gorm:"not null;column:restored_at;default:now()"`
	Status         string `gorm:"not null;column:status;default:'complete'"`
}

func (tenantBaseline) TableName() string { return "tenant_baselines" }

// Migration auto-creates/updates the tenant_baselines table. AutoMigrate adds
// the status column as NOT NULL DEFAULT 'complete', which Postgres backfills
// existing rows with atomically — and pre-existing rows only ever existed on a
// fully-successful restore, so StatusComplete is correct for them. The explicit
// UPDATE is a defensive backstop for any row that somehow predates the default
// (e.g. a manual insert); reconciliation must never re-run an already-good
// tenant.
func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&tenantBaseline{}); err != nil {
		return err
	}
	return db.Exec(`UPDATE tenant_baselines SET status = ? WHERE status IS NULL OR status = ''`, StatusComplete).Error
}
