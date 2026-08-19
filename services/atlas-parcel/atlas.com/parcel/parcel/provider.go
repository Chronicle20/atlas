package parcel

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ById returns the parcel with the given id. Tenant scoping is applied by
// the atlas-database tenant callback from the context on db — this provider
// must NOT add a manual tenant_id predicate.
func ById(id uuid.UUID) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		var e Entity
		err := db.Where("id = ?", id).First(&e).Error
		if err != nil {
			return model.ErrorProvider[Model](err)
		}
		return model.Map(Make)(model.FixedProvider(e))
	}
}

// ByRecipient returns a recipient's parcels in a world with the given
// status. Backed by idx_parcels_recipient (tenant_id, recipient_id,
// status) — the WHERE clause is an explicit name-keyed map so world 0 / a
// zero-valued status is never silently dropped by GORM's struct-condition
// elision.
func ByRecipient(recipientId uint32, worldId world.Id, status string) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		var results []Entity
		err := db.Where(map[string]interface{}{
			"recipient_id": recipientId,
			"world_id":     byte(worldId),
			"status":       status,
		}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Model](err)
		}
		return model.SliceMap(Make)(model.FixedProvider(results))()
	}
}

// BySender returns a sender's parcels with the given status. Backed by
// idx_parcels_sender (tenant_id, sender_id, status).
func BySender(senderId uint32, status string) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		var results []Entity
		err := db.Where(map[string]interface{}{
			"sender_id": senderId,
			"status":    status,
		}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Model](err)
		}
		return model.SliceMap(Make)(model.FixedProvider(results))()
	}
}

// ReceivableByRecipient returns a recipient's pending parcels in a world
// whose ReceivableAt has passed as of now — the mailbox-open query.
func ReceivableByRecipient(recipientId uint32, worldId world.Id, now time.Time) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		var results []Entity
		err := db.Where(map[string]interface{}{
			"recipient_id": recipientId,
			"world_id":     byte(worldId),
			"status":       StatusPending,
		}).Where("receivable_at <= ?", now).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Model](err)
		}
		return model.SliceMap(Make)(model.FixedProvider(results))()
	}
}
