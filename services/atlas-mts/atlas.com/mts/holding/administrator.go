package holding

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

// GetById is the exported provider wrapper: it resolves a holding by surrogate
// id, mapping the entity to the immutable Model.
func GetById(id string) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		return model.Map(modelFromEntity)(getById(id)(db))
	}
}

// GetAll resolves every holding visible to the request's tenant.
func GetAll() database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getAll()(db))()
	}
}

// CreateHolding assigns a fresh surrogate id, persists an explicit-column row,
// and returns the stored Model.
func CreateHolding(db *gorm.DB, m Model) (Model, error) {
	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}
	createdAt := m.CreatedAt()
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	e := entity{
		Id:            id,
		TenantId:      m.TenantId(),
		WorldId:       byte(m.WorldId()),
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
	result := db.Where(&entity{Id: parseId(id)}).Delete(&entity{})
	return result.RowsAffected, result.Error
}
