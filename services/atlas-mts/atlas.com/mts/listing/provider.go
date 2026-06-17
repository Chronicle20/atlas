package listing

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// BrowseFilter carries the optional public-browse filters. The zero value of a
// pointer/empty string means "do not constrain on this column"; world_id and
// state are always applied (they are required positional args to getBrowse).
type BrowseFilter struct {
	Category    string
	SubCategory string
	SaleType    SaleType
	ItemId      uint32
	SellerName  string
	Page        int
	PageSize    int
}

// DefaultPageSize is the browse page size when the caller does not specify one.
const DefaultPageSize = 16

// getBrowse returns the listings for a world filtered by state and the optional
// filter set, paginated.
//
// The WHERE clause is built incrementally — a column is only constrained when
// the caller actually provided a filter for it. world_id and state are always
// applied via an explicit name-keyed map rather than a struct condition: GORM's
// struct-condition Where elides zero-valued fields, so a struct condition would
// silently drop the world_id filter for world 0 (a valid world.Id, since
// world.Id is a byte) and return cross-world rows. The conditional clauses below
// likewise never use struct conditions, so a zero-valued optional filter is
// simply omitted rather than matched against zero.
func getBrowse(worldId world.Id, state State, f BrowseFilter) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity

		q := db.Where(map[string]interface{}{
			"world_id": byte(worldId),
			"state":    string(state),
		})
		if f.Category != "" {
			q = q.Where("category = ?", f.Category)
		}
		if f.SubCategory != "" {
			q = q.Where("sub_category = ?", f.SubCategory)
		}
		if f.SaleType != "" {
			q = q.Where("sale_type = ?", string(f.SaleType))
		}
		if f.ItemId != 0 {
			q = q.Where("template_id = ?", f.ItemId)
		}
		if f.SellerName != "" {
			q = q.Where("seller_name = ?", f.SellerName)
		}

		pageSize := f.PageSize
		if pageSize <= 0 {
			pageSize = DefaultPageSize
		}
		page := f.Page
		if page < 0 {
			page = 0
		}
		q = q.Limit(pageSize).Offset(page * pageSize)

		err := q.Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

// getActiveCountBySeller returns the number of active listings owned by a seller.
//
// The WHERE clause uses an explicit name-keyed map rather than a struct
// condition: GORM's struct-condition Where elides zero-valued fields, so a
// struct condition would silently drop the seller_id filter for seller 0 and
// over-count. state is constrained to active so cancelled/sold/expired rows do
// not count against the per-character cap.
func getActiveCountBySeller(sellerId uint32) func(db *gorm.DB) (int64, error) {
	return func(db *gorm.DB) (int64, error) {
		var count int64
		err := db.Model(&entity{}).
			Where(map[string]interface{}{
				"seller_id": sellerId,
				"state":     string(StateActive),
			}).
			Count(&count).Error
		if err != nil {
			return 0, err
		}
		return count, nil
	}
}

// getExpiredActive returns the active listings whose auction window has closed
// (ends_at IS NOT NULL AND ends_at < now). Fixed-price listings (ends_at NULL)
// are excluded — only auctions expire. The caller controls tenant scoping via the
// db's context: the periodic sweep passes a WithoutTenantFilter context plus an
// explicit tenant_id so the discovery is cross-tenant, while a tenant-scoped
// context narrows it to one tenant. An optional limit bounds the batch (0 = no
// limit); the sweep logs anything left for the next tick rather than silently
// truncating (NFR 8.3).
func getExpiredActive(now time.Time, limit int) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		q := db.Where("state = ?", string(StateActive)).
			Where("ends_at IS NOT NULL").
			Where("ends_at < ?", now)
		if limit > 0 {
			q = q.Limit(limit)
		}
		if err := q.Find(&results).Error; err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

// countExpiredActive returns the total number of expired active auction listings
// matching the same predicate as getExpiredActive, ignoring any batch limit. The
// sweep compares this against the number it processed to log the deferred tail.
func countExpiredActive(now time.Time) func(db *gorm.DB) (int64, error) {
	return func(db *gorm.DB) (int64, error) {
		var count int64
		err := db.Model(&entity{}).
			Where("state = ?", string(StateActive)).
			Where("ends_at IS NOT NULL").
			Where("ends_at < ?", now).
			Count(&count).Error
		if err != nil {
			return 0, err
		}
		return count, nil
	}
}

func modelFromEntity(e entity) (Model, error) {
	b := NewBuilder(e.TenantId, world.Id(e.WorldId), e.SellerId).
		SetId(e.Id).
		SetSellerName(e.SellerName).
		SetSaleType(SaleType(e.SaleType)).
		SetState(State(e.State)).
		SetTemplateId(e.TemplateId).
		SetQuantity(e.Quantity).
		SetStrength(e.Strength).
		SetDexterity(e.Dexterity).
		SetIntelligence(e.Intelligence).
		SetLuck(e.Luck).
		SetHP(e.HP).
		SetMP(e.MP).
		SetWeaponAttack(e.WeaponAttack).
		SetMagicAttack(e.MagicAttack).
		SetWeaponDefense(e.WeaponDefense).
		SetMagicDefense(e.MagicDefense).
		SetAccuracy(e.Accuracy).
		SetAvoidability(e.Avoidability).
		SetHands(e.Hands).
		SetSpeed(e.Speed).
		SetJump(e.Jump).
		SetSlots(e.Slots).
		SetLevel(e.Level).
		SetItemLevel(e.ItemLevel).
		SetItemExp(e.ItemExp).
		SetRingId(e.RingId).
		SetViciousCount(e.ViciousCount).
		SetFlags(e.Flags).
		SetListValue(e.ListValue).
		SetBuyNowPrice(e.BuyNowPrice).
		SetCommissionRate(e.CommissionRate).
		SetCategory(e.Category).
		SetSubCategory(e.SubCategory).
		SetEndsAt(e.EndsAt).
		SetCurrentBid(e.CurrentBid).
		SetHighBidderId(e.HighBidderId).
		SetMinIncrement(e.MinIncrement).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt)
	return b.Build()
}
