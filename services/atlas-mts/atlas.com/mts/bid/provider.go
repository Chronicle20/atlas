package bid

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getAll() database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{})
	}
}

func getById(id string) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{Id: parseId(id)})
	}
}

// getByListingId returns the bids placed on an auction listing.
//
// The filter is built as an explicit name-keyed map rather than a struct
// condition: GORM's struct-condition Where elides zero-valued fields, so the
// map form forces every filter column into the WHERE clause regardless of the
// value.
func getByListingId(listingId uuid.UUID) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(map[string]interface{}{
			"listing_id": listingId,
		}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

func modelFromEntity(e entity) (Model, error) {
	return NewBuilder(e.TenantId, e.ListingId, e.BidderId).
		SetId(e.Id).
		SetBidderAccountId(e.BidderAccountId).
		SetAmount(e.Amount).
		SetEscrowTxnId(e.EscrowTxnId).
		SetState(State(e.State)).
		SetCreatedAt(e.CreatedAt).
		Build()
}
