package wish

import (
	"errors"
	"fmt"
	"time"

	"atlas-mts/serial"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

// GetById is the exported provider wrapper: it resolves a wish entry by
// surrogate id, mapping the entity to the immutable Model.
func GetById(id string) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		return model.Map(modelFromEntity)(getById(id)(db))
	}
}

// GetBySerial is the exported provider wrapper: it resolves a wish entry by its
// per-(tenant, world) ITC serial (the client's nITCSN), mapping the entity to
// the immutable Model. This backs the CANCEL_WISH serial -> wish resolution.
func GetBySerial(worldId world.Id, sn uint32) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		return model.Map(modelFromEntity)(getBySerial(worldId, sn)(db))
	}
}

// GetAll resolves every wish entry visible to the request's tenant.
func GetAll() database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getAll()(db))()
	}
}

// CreateWish is the idempotent wish-create: it enforces the "one wish per
// (tenant, world, character, item)" invariant. If the character already wishes
// for that item in that world, the EXISTING entry is returned unchanged and NO
// new serial is consumed; otherwise a fresh surrogate id and a per-(tenant,
// world) ITC serial (drawn from the shared `serial` counter, the same one
// listings/holdings use) are assigned and the row is inserted.
//
// The serial is drawn here — at the INSERT choke point — using the SAME db
// handle the caller passes, so the serial advance and the row insert commit or
// roll back together within the caller's transaction (handleRegisterWish wraps
// this in an ExecuteTransaction). The existence check is performed first so a
// duplicate REGISTER_WISH short-circuits before drawing a serial.
func CreateWish(db *gorm.DB, m Model) (Model, error) {
	// Idempotency: a wish already held for (tenant, world, character, item)
	// returns unchanged. The unique index backs this at the DB level; the
	// explicit pre-check keeps the serial draw out of the duplicate path.
	if existing, err := getByCharacterItem(m.WorldId(), m.CharacterId(), m.ItemId(), m.Type())(db)(); err == nil {
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Model{}, err
	}

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
		Id:          id,
		TenantId:    m.TenantId(),
		WorldId:     byte(m.WorldId()),
		Serial:      sn,
		CharacterId: m.CharacterId(),
		ItemId:      m.ItemId(),
		Type:        m.Type(),
		Price:       m.Price(),
		CreatedAt:   createdAt,
	}
	if err := db.Create(&e).Error; err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}

// DeleteWish hard-deletes the wish entry by id, returning the number of rows
// affected (1 on a delete, 0 if it was already gone). Wish entries are not
// custody, so a hard delete is appropriate here. The tenant callback scopes the
// write to the request's tenant.
func DeleteWish(db *gorm.DB, id string) (int64, error) {
	wid := parseId(id)
	if wid == uuid.Nil {
		// Guard against the GORM zero-value struct-condition elision: a uuid.Nil
		// Id condition would vanish from the WHERE, degrading the delete to a
		// tenant-wide wipe. Reject the malformed id before touching the DB.
		return 0, fmt.Errorf("invalid wish id %q", id)
	}
	// The map-keyed WHERE forces the id into the query: a struct condition would
	// elide a zero-valued id (defense-in-depth alongside the guard above).
	result := db.Where(map[string]interface{}{"id": wid}).Delete(&entity{})
	return result.RowsAffected, result.Error
}
