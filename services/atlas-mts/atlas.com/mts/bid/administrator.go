package bid

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

// GetById is the exported provider wrapper: it resolves a bid by surrogate id,
// mapping the entity to the immutable Model.
func GetById(id string) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		return model.Map(modelFromEntity)(getById(id)(db))
	}
}

// GetByListingId is the exported provider wrapper: it resolves every bid placed on
// an auction listing, mapping each entity to the immutable Model. The listing
// processor uses it to find a bidder's held bid (for the outbid release and the
// settle-at-expiry win mark) within the same DB handle/transaction.
func GetByListingId(listingId uuid.UUID) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getByListingId(listingId)(db))()
	}
}

// GetAll resolves every bid visible to the request's tenant.
func GetAll() database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getAll()(db))()
	}
}

// CreateBid assigns a fresh surrogate id, persists the row, and returns the
// stored Model.
func CreateBid(db *gorm.DB, m Model) (Model, error) {
	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}
	createdAt := m.CreatedAt()
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	e := entity{
		Id:              id,
		TenantId:        m.TenantId(),
		ListingId:       m.ListingId(),
		BidderId:        m.BidderId(),
		BidderAccountId: m.BidderAccountId(),
		Amount:          m.Amount(),
		EscrowTxnId:     m.EscrowTxnId(),
		State:           string(m.State()),
		CreatedAt:       createdAt,
	}
	if err := db.Create(&e).Error; err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}

// UpdateState performs the race-safe conditional transition. It updates the row
// only when its current state equals `from`, returning the number of rows
// affected: 1 on a successful transition, 0 if another writer already moved the
// row out of `from`. The tenant callback scopes the write to the request's
// tenant.
func UpdateState(db *gorm.DB, id string, from State, to State) (int64, error) {
	result := db.Model(&entity{}).
		Where(&entity{Id: parseId(id), State: string(from)}).
		Updates(map[string]interface{}{
			"state": string(to),
		})
	return result.RowsAffected, result.Error
}
