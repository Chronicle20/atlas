package transaction

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateTransaction assigns a fresh surrogate id (if none is set), persists an
// explicit-column row, and returns the stored Model. It uses the SAME db handle
// the caller passes, so it composes inside a caller-provided transaction (the
// settle handler records both the buyer purchase row and the seller sale row in
// the same ExecuteTransaction as the holding move).
func CreateTransaction(db *gorm.DB, m Model) (Model, error) {
	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}
	createdAt := m.CreatedAt()
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	e := entity{
		Id:             id,
		TenantId:       m.TenantId(),
		WorldId:        byte(m.WorldId()),
		CharacterId:    m.CharacterId(),
		CounterpartyId: m.CounterpartyId(),
		ItemId:         m.ItemId(),
		Quantity:       m.Quantity(),
		TotalPrice:     m.TotalPrice(),
		Kind:           m.Kind(),
		CreatedAt:      createdAt,
	}
	if err := db.Create(&e).Error; err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}
