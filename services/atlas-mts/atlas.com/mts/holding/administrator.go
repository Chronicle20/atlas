package holding

import (
	"atlas-mts/serial"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// parseId converts a string id into a uuid, returning uuid.Nil on a malformed
// value so a bad path param degrades to a not-found query rather than panicking.
func parseId(id string) uuid.UUID {
	u, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil
	}
	return u
}

// GetById is the exported provider wrapper: it resolves a holding by surrogate
// id, mapping the entity to the immutable Model.
func GetById(id string) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		return model.Map(modelFromEntity)(getById(id)(db))
	}
}

// GetBySerial is the exported provider wrapper: it resolves a holding by its
// per-(tenant, world) ITC serial (the client's nITCSN), mapping the entity to the
// immutable Model.
func GetBySerial(worldId world.Id, sn uint32) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		return model.Map(modelFromEntity)(getBySerial(worldId, sn)(db))
	}
}

// GetAll resolves every holding visible to the request's tenant.
func GetAll() database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getAll()(db))()
	}
}

// CreateHolding assigns a fresh surrogate id, draws the next per-(tenant, world)
// ITC serial (the client's nITCSN, shared with listings), persists an
// explicit-column row, and returns the stored Model.
//
// The serial is drawn here — at the INSERT choke point — using the SAME db handle
// the caller passes. Every production caller (the cancel/expire seller-holding
// transition in the listing processor and the MtsMoveListingToHolding settle
// handler) invokes CreateHolding only AFTER a guard inside an ExecuteTransaction
// (a conditional active->terminal UpdateState that yields 0 rows on replay, or an
// id-existence check), so a replayed move/cancel/expire short-circuits before
// reaching here and never consumes a serial; the serial draw and the row insert
// then commit or roll back together within that one transaction.
func CreateHolding(db *gorm.DB, m Model) (Model, error) {
	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}
	createdAt := m.CreatedAt()
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	sn, err := serial.Next(db, m.TenantId(), m.WorldId())
	if err != nil {
		return Model{}, err
	}

	e := entity{
		Id:            id,
		TenantId:      m.TenantId(),
		WorldId:       byte(m.WorldId()),
		Serial:        sn,
		OwnerId:       m.OwnerId(),
		Origin:        string(m.Origin()),
		TemplateId:    m.TemplateId(),
		Quantity:      m.Quantity(),
		Strength:      m.Strength(),
		Dexterity:     m.Dexterity(),
		Intelligence:  m.Intelligence(),
		Luck:          m.Luck(),
		HP:            m.HP(),
		MP:            m.MP(),
		WeaponAttack:  m.WeaponAttack(),
		MagicAttack:   m.MagicAttack(),
		WeaponDefense: m.WeaponDefense(),
		MagicDefense:  m.MagicDefense(),
		Accuracy:      m.Accuracy(),
		Avoidability:  m.Avoidability(),
		Hands:         m.Hands(),
		Speed:         m.Speed(),
		Jump:          m.Jump(),
		Slots:         m.Slots(),
		Level:         m.Level(),
		ItemLevel:     m.ItemLevel(),
		ItemExp:       m.ItemExp(),
		RingId:        m.RingId(),
		ViciousCount:  m.ViciousCount(),
		Flags:         m.Flags(),
		Owner:         m.Owner(),
		CreatedAt:     createdAt,
	}
	if err := db.Create(&e).Error; err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}

// SoftDelete soft-deletes the holding by id, returning the number of rows
// affected. Take-home is idempotent: the first call soft-deletes the row (1
// row), a second call affects 0 rows because the row is already gone from the
// default (non-deleted) scope. The tenant callback scopes the write to the
// request's tenant.
func SoftDelete(db *gorm.DB, id string) (int64, error) {
	hid := parseId(id)
	if hid == uuid.Nil {
		// Guard against the GORM zero-value struct-condition elision: a uuid.Nil
		// Id condition would vanish, degrading the delete to a tenant-wide wipe.
		return 0, fmt.Errorf("invalid holding id %q", id)
	}
	result := db.Where(map[string]interface{}{"id": hid}).Delete(&entity{})
	return result.RowsAffected, result.Error
}

// Restore un-soft-deletes the holding by id (clears deleted_at), returning the
// number of rows affected. It is the inverse of SoftDelete and is idempotent:
// restoring an already-live row clears nothing (0 rows) and is still success.
// Unscoped is required so the UPDATE can see the soft-deleted row.
func Restore(db *gorm.DB, id string) (int64, error) {
	hid := parseId(id)
	if hid == uuid.Nil {
		// Guard against the GORM zero-value struct-condition elision: a uuid.Nil
		// Id condition would vanish, degrading the update to a tenant-wide restore.
		return 0, fmt.Errorf("invalid holding id %q", id)
	}
	result := db.Unscoped().Model(&entity{}).
		Where(map[string]interface{}{"id": hid}).
		Where("deleted_at IS NOT NULL").
		Update("deleted_at", nil)
	return result.RowsAffected, result.Error
}
