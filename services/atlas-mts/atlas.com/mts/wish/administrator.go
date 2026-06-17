package wish

import (
	"time"

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

// GetAll resolves every wish entry visible to the request's tenant.
func GetAll() database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getAll()(db))()
	}
}

// CreateWish assigns a fresh surrogate id, persists the row, and returns the
// stored Model.
func CreateWish(db *gorm.DB, m Model) (Model, error) {
	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}
	createdAt := m.CreatedAt()
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	e := entity{
		Id:          id,
		TenantId:    m.TenantId(),
		CharacterId: m.CharacterId(),
		ItemId:      m.ItemId(),
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
	result := db.Where(&entity{Id: parseId(id)}).Delete(&entity{})
	return result.RowsAffected, result.Error
}
