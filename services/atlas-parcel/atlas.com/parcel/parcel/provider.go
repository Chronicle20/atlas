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
		return model.Map(Make)(database.Query[Entity](db.Where("id = ?", id), &Entity{}))
	}
}

// ByRecipient returns a recipient's parcels in a world with the given
// status. Backed by idx_parcels_recipient (tenant_id, recipient_id,
// status) — the WHERE clause is an explicit name-keyed map so world 0 / a
// zero-valued status is never silently dropped by GORM's struct-condition
// elision.
func ByRecipient(recipientId uint32, worldId world.Id, status string) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(Make)(database.SliceQuery[Entity](db.Where(map[string]interface{}{
			"recipient_id": recipientId,
			"world_id":     byte(worldId),
			"status":       status,
		}), &Entity{}))()
	}
}

// BySender returns a sender's parcels with the given status. Backed by
// idx_parcels_sender (tenant_id, sender_id, status).
func BySender(senderId uint32, status string) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(Make)(database.SliceQuery[Entity](db.Where(map[string]interface{}{
			"sender_id": senderId,
			"status":    status,
		}), &Entity{}))()
	}
}

// ReceivableByRecipient returns a recipient's pending parcels in a world
// whose ReceivableAt has passed as of now — the mailbox-open query.
func ReceivableByRecipient(recipientId uint32, worldId world.Id, now time.Time) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(Make)(database.SliceQuery[Entity](db.Where(map[string]interface{}{
			"recipient_id": recipientId,
			"world_id":     byte(worldId),
			"status":       StatusPending,
		}).Where("receivable_at <= ?", now), &Entity{}))()
	}
}

// ReceivableByRecipientAnyWorld is ReceivableByRecipient without the
// world_id predicate — a tenant is multi-world and a character's inbound
// parcels can have been sent within any of them, so "does this character
// have a receivable inbound parcel" (Processor.HasInFlight's inbound half,
// design §9.1 / gate 12) must not be scoped to a single world. Still backed
// by idx_parcels_recipient (tenant_id, recipient_id, status); dropping the
// world_id residual filter does not lose the index.
func ReceivableByRecipientAnyWorld(recipientId uint32, now time.Time) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(Make)(database.SliceQuery[Entity](db.Where(map[string]interface{}{
			"recipient_id": recipientId,
			"status":       StatusPending,
		}).Where("receivable_at <= ?", now), &Entity{}))()
	}
}
