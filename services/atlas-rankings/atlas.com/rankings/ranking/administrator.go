package ranking

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const upsertBatchSize = 500

// upsertBatch inserts or updates ranking rows for the given tenant, 500 rows
// per batch. Conflicts are detected on (tenant_id, character_id) — a
// character keeps its row identity (Id) across cycles while its rank fields
// are refreshed. Callers must invoke this through db.WithContext(ctx) so the
// tenant callback stamps/validates TenantId consistently with the rest of
// the codebase; TenantId is set explicitly here as well because Create
// bypasses the query-side tenant filter.
func upsertBatch(db *gorm.DB, tenantId uuid.UUID, entities []Entity) error {
	if len(entities) == 0 {
		return nil
	}
	for i := range entities {
		entities[i].TenantId = tenantId
		if entities[i].Id == uuid.Nil {
			entities[i].Id = uuid.New()
		}
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"world_id", "job_category",
			"overall_rank", "overall_rank_move",
			"job_rank", "job_rank_move",
			"computed_at",
		}),
	}).CreateInBatches(&entities, upsertBatchSize).Error
}

// pruneBefore removes rows not restamped by the current cycle — deleted
// characters and characters that became GM. Tenant scoping relies on the
// GORM delete callback registered by database.RegisterTenantCallbacks
// against the context-bearing db handle passed in by the caller; this
// function must never be called with a db that was not derived via
// db.WithContext(ctx). This is the highest-risk operation in this
// package — a bypassed callback would delete every tenant's stale rows,
// not just the caller's.
//
// That callback is fail-open by design (it returns without adding a WHERE
// clause when tenant.FromContext errors — see
// libs/atlas-database/tenant_scope.go), so pruneBefore fails closed itself
// by resolving the tenant from the same context before issuing the DELETE.
// A caller that passes a db handle with no resolvable tenant (e.g. a raw
// handle from database.Connect rather than db.WithContext(ctx)) gets an
// error instead of an unscoped DELETE across every tenant's rows.
func pruneBefore(db *gorm.DB, cycleTime time.Time) error {
	if _, err := tenant.FromContext(db.Statement.Context)(); err != nil {
		return fmt.Errorf("pruneBefore: no tenant resolvable from context, refusing unscoped delete: %w", err)
	}
	return db.Where("computed_at < ?", cycleTime).Delete(&Entity{}).Error
}

// startCycle records the beginning of a recompute cycle, upserting on the
// unique tenant_id column. LastCompletedAt/CharactersRanked/DurationMs are
// left untouched by this call (DoUpdates covers only last_started_at) so a
// crash between startCycle and completeCycle preserves the previous cycle's
// completion stats for observability.
func startCycle(db *gorm.DB, tenantId uuid.UUID, startedAt time.Time) error {
	e := CycleEntity{
		TenantId:      tenantId,
		Id:            uuid.New(),
		LastStartedAt: startedAt,
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"last_started_at": startedAt}),
	}).Create(&e).Error
}

// completeCycle records the end of a recompute cycle for the given tenant.
// startCycle must have run first (in this cycle or a previous one) so the
// row exists; this only updates it.
func completeCycle(db *gorm.DB, tenantId uuid.UUID, completedAt time.Time, charactersRanked uint32, durationMs uint32) error {
	return db.Model(&CycleEntity{}).
		Where("tenant_id = ?", tenantId).
		Updates(map[string]interface{}{
			"last_completed_at": completedAt,
			"characters_ranked": charactersRanked,
			"duration_ms":       durationMs,
		}).Error
}
