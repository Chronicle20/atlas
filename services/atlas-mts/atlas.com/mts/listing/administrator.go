package listing

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

// GetById is the exported provider wrapper: it resolves a listing by surrogate
// id, mapping the entity to the immutable Model.
func GetById(id string) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		return model.Map(modelFromEntity)(getById(id)(db))
	}
}

// GetAll resolves every listing visible to the request's tenant.
func GetAll() database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getAll()(db))()
	}
}

// GetExpiredActive resolves the active auction listings whose ends_at has passed
// (not null, < now), capped at limit (0 = uncapped). Tenant scoping is the
// caller's responsibility via the db context: the expiration sweep passes a
// WithoutTenantFilter context to discover expired listings across every tenant.
func GetExpiredActive(now time.Time, limit int) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getExpiredActive(now, limit)(db))()
	}
}

// CountExpiredActive returns the total number of expired active auction listings
// (ignoring any batch limit), so the sweep can log how many it deferred to the
// next tick rather than silently truncating (NFR 8.3).
func CountExpiredActive(now time.Time) func(db *gorm.DB) (int64, error) {
	return countExpiredActive(now)
}

// CreateListing assigns a fresh surrogate id, persists an explicit-column row,
// and returns the stored Model.
func CreateListing(db *gorm.DB, m Model) (Model, error) {
	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}
	now := time.Now()
	createdAt := m.CreatedAt()
	if createdAt.IsZero() {
		createdAt = now
	}

	e := entity{
		Id:             id,
		TenantId:       m.TenantId(),
		WorldId:        byte(m.WorldId()),
		SellerId:       m.SellerId(),
		SellerName:     m.SellerName(),
		SaleType:       string(m.SaleType()),
		State:          string(m.State()),
		TemplateId:     m.TemplateId(),
		Quantity:       m.Quantity(),
		Strength:       m.Strength(),
		Dexterity:      m.Dexterity(),
		Intelligence:   m.Intelligence(),
		Luck:           m.Luck(),
		HP:             m.HP(),
		MP:             m.MP(),
		WeaponAttack:   m.WeaponAttack(),
		MagicAttack:    m.MagicAttack(),
		WeaponDefense:  m.WeaponDefense(),
		MagicDefense:   m.MagicDefense(),
		Accuracy:       m.Accuracy(),
		Avoidability:   m.Avoidability(),
		Hands:          m.Hands(),
		Speed:          m.Speed(),
		Jump:           m.Jump(),
		Slots:          m.Slots(),
		Level:          m.Level(),
		ItemLevel:      m.ItemLevel(),
		ItemExp:        m.ItemExp(),
		RingId:         m.RingId(),
		ViciousCount:   m.ViciousCount(),
		Flags:          m.Flags(),
		ListValue:      m.ListValue(),
		BuyNowPrice:    m.BuyNowPrice(),
		CommissionRate: m.CommissionRate(),
		Category:       m.Category(),
		SubCategory:    m.SubCategory(),
		EndsAt:         m.EndsAt(),
		CurrentBid:     m.CurrentBid(),
		HighBidderId:   m.HighBidderId(),
		MinIncrement:   m.MinIncrement(),
		CreatedAt:      createdAt,
		UpdatedAt:      now,
	}
	if err := db.Create(&e).Error; err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}

// UpdateState performs the race-safe conditional transition. It updates the row
// only when its current state equals `from`, returning the number of rows
// affected: 1 on a successful transition, 0 if another writer already moved the
// row out of `from` (the cancel-vs-buy race). The tenant callback scopes the
// write to the request's tenant.
func UpdateState(db *gorm.DB, id string, from State, to State) (int64, error) {
	result := db.Model(&entity{}).
		Where(&entity{Id: parseId(id), State: string(from)}).
		Updates(map[string]interface{}{
			"state":      string(to),
			"updated_at": time.Now(),
		})
	return result.RowsAffected, result.Error
}

// UpdateAuction updates the live auction fields (current bid, high bidder, and
// the optional end time) within a transaction. Used by the bid path.
func UpdateAuction(db *gorm.DB, id string, currentBid uint32, highBidderId uint32, endsAt *time.Time) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Model(&entity{}).
			Where(&entity{Id: parseId(id)}).
			Updates(map[string]interface{}{
				"current_bid":    currentBid,
				"high_bidder_id": highBidderId,
				"ends_at":        endsAt,
				"updated_at":     time.Now(),
			}).Error
	})
}
