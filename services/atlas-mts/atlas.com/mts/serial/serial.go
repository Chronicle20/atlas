// Package serial implements the persistent per-(tenant, world) monotonic ITC
// serial (the client's nITCSN). A single shared counter per (tenant, world) is
// drawn from for BOTH listings and holdings, so a serial maps to exactly one
// listing OR one holding within a world (no cross-table collision). The serial
// is the uint32 the MTS wire uses to address listings/holdings (serverbound
// buy/cancel/bid/take-home carry it; clientbound browse emits MtsItem.itcSn).
package serial

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// entity is the per-(tenant, world) counter row. NextSerial holds the LAST
// assigned serial; Next advances it (NextSerial = NextSerial + 1) and returns
// the new value, so the first serial assigned in a (tenant, world) is 1.
//
// The composite primary key (tenant_id, world_id) makes the counter row unique
// per world and lets the seed upsert use ON CONFLICT DO NOTHING. tenant_id is a
// real column so the tenant query/create callbacks scope it automatically — but
// Next deliberately drives every counter access through an explicit name-keyed
// WHERE rather than relying on the callback, because world 0 is a valid world
// and a struct condition would elide it.
type entity struct {
	TenantId   uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;primaryKey"`
	WorldId    byte      `gorm:"column:world_id;not null;primaryKey"`
	NextSerial uint32    `gorm:"column:next_serial;not null"`
}

func (entity) TableName() string {
	return "mts_serials"
}

// Migration creates the mts_serials counter table. Additive — a new table, so
// AutoMigrate alone produces the correct shape.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

// Next atomically assigns and returns the next serial for (tenantId, worldId).
//
// It MUST be called inside the same transaction as the row INSERT it is serving
// (the creation administrators do exactly this), so the counter advance and the
// row insert commit or roll back together — a rolled-back create never consumes
// a serial.
//
// Atomicity / concurrency guarantee (sqlite + postgres):
//
//	Step 1 seeds the counter row with NextSerial=0 via INSERT ... ON CONFLICT DO
//	NOTHING, so a missing (tenant, world) starts at 0 and a present one is left
//	untouched.
//	Step 2 issues UPDATE mts_serials SET next_serial = next_serial + 1 WHERE
//	(tenant_id, world_id) = (...). This is a single read-modify-write statement;
//	the database evaluates next_serial + 1 against the CURRENT committed/locked
//	value and takes a row write-lock held until the enclosing transaction
//	commits. In postgres a concurrent Next on the same (tenant, world) blocks on
//	that row lock until this tx commits, then reads the incremented value — no
//	two transactions can compute the same next_serial. In sqlite, writes are
//	serialized at the database level (a write transaction holds an exclusive
//	lock), so the increment is likewise serialized. The +1 is never computed in
//	application code, so there is no read-then-write race window in either DB.
//	Step 3 SELECTs the just-incremented value within the same locked tx and
//	returns it as the assigned serial.
//
// tenant_id is NOT taken from context here — it is passed explicitly so the
// caller (which already holds the row's tenant) controls scoping, and so the
// cross-tenant ticker paths can advance the correct counter. All three steps use
// explicit name-keyed WHERE clauses (never struct conditions), so world 0 — a
// valid world.Id — is matched rather than elided.
func Next(db *gorm.DB, tenantId uuid.UUID, worldId world.Id) (uint32, error) {
	wid := byte(worldId)

	// Step 1: seed the counter row at 0 if it does not yet exist. ON CONFLICT DO
	// NOTHING leaves an existing counter untouched.
	seed := entity{TenantId: tenantId, WorldId: wid, NextSerial: 0}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
		return 0, err
	}

	// Step 2: atomic read-modify-write increment under a row write-lock.
	if err := db.Model(&entity{}).
		Where(map[string]interface{}{
			"tenant_id": tenantId,
			"world_id":  wid,
		}).
		UpdateColumn("next_serial", gorm.Expr("next_serial + 1")).Error; err != nil {
		return 0, err
	}

	// Step 3: read the just-assigned value within the same locked tx.
	var assigned entity
	if err := db.
		Where(map[string]interface{}{
			"tenant_id": tenantId,
			"world_id":  wid,
		}).
		First(&assigned).Error; err != nil {
		return 0, err
	}
	return assigned.NextSerial, nil
}
