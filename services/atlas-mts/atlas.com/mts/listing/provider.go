package listing

import (
	"time"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// getBySerial resolves a listing by its per-(tenant, world) ITC serial (the
// client's nITCSN). The WHERE clause is an explicit name-keyed map rather than a
// struct condition: GORM's struct-condition Where elides zero-valued fields, so a
// struct condition would silently drop the world_id filter for world 0 (a valid
// world.Id, since world.Id is a byte) and resolve the wrong row. tenant scoping is
// applied by the tenant query callback from the db's context.
func getBySerial(worldId world.Id, sn uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		var result entity
		err := db.Where(map[string]interface{}{
			"world_id": byte(worldId),
			"serial":   sn,
		}).First(&result).Error
		if err != nil {
			return model.ErrorProvider[entity](err)
		}
		return model.FixedProvider(result)
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
	// ItemIds restricts the browse to this set of template ids (template_id IN (?)).
	// Used by the marketplace name search, which resolves a search term to its
	// matching item template ids and filters the listings on them.
	ItemIds []uint32
	Serial  uint32
	// Serials restricts the browse to this set of ITC serials (serial IN (?)). Used
	// by the Cart, which resolves its entries' favorited listing serials in ONE
	// browse instead of a per-entry GetBySerial (avoids the N+1 on every re-push).
	Serials         []uint32
	SellerId        uint32
	ExcludeSellerId uint32
	SellerName      string
	// OfferWishSerial filters to offer listings on a specific want-ad.
	OfferWishSerial uint32
	// ExcludeOffers omits sale_type=offer rows from a public browse.
	ExcludeOffers bool
	Page          int
	PageSize      int
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
// browseFilterQuery applies the world/state scope plus every optional browse
// filter to db, returning the narrowed query WITHOUT paging. getBrowse (page
// slice) and countBrowse (total match count) share it so the two can never
// drift — a filter that narrows the page must narrow the count identically, or
// the client's total/last-page would lie.
func browseFilterQuery(db *gorm.DB, worldId world.Id, state State, f BrowseFilter) *gorm.DB {
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
	if len(f.ItemIds) > 0 {
		q = q.Where("template_id IN ?", f.ItemIds)
	}
	if f.Serial != 0 {
		q = q.Where("serial = ?", f.Serial)
	}
	if len(f.Serials) > 0 {
		q = q.Where("serial IN ?", f.Serials)
	}
	if f.SellerId != 0 {
		q = q.Where("seller_id = ?", f.SellerId)
	}
	if f.ExcludeSellerId != 0 {
		q = q.Where("seller_id <> ?", f.ExcludeSellerId)
	}
	if f.SellerName != "" {
		q = q.Where("seller_name = ?", f.SellerName)
	}
	if f.OfferWishSerial != 0 {
		q = q.Where("offer_wish_serial = ?", f.OfferWishSerial)
	}
	if f.ExcludeOffers {
		q = q.Where("sale_type <> ?", "offer")
	}
	return q
}

// countBrowse returns the TOTAL number of listings matching the browse filters,
// ignoring paging — the value the client needs to render a real total / last
// page (getBrowse only returns one page slice). Shares browseFilterQuery with
// getBrowse so the count and the page always agree.
func countBrowse(worldId world.Id, state State, f BrowseFilter) func(db *gorm.DB) (int64, error) {
	return func(db *gorm.DB) (int64, error) {
		var total int64
		err := browseFilterQuery(db, worldId, state, f).Model(&entity{}).Count(&total).Error
		return total, err
	}
}

func getBrowse(worldId world.Id, state State, f BrowseFilter) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity

		q := browseFilterQuery(db, worldId, state, f)

		// PageSize < 0 disables paging entirely and returns every filtered
		// row: the channel derives the client's categoryItemCnt total from
		// the full set (the v83 page selector is ceil(total/16) —
		// CITCWnd_List::ChangeCategorySub, 0x5BDD12 — so the total must span
		// all pages) and slices the 16-item page window itself.
		// PageSize == 0 keeps the DefaultPageSize window.
		if f.PageSize >= 0 {
			pageSize := f.PageSize
			if pageSize == 0 {
				pageSize = DefaultPageSize
			}
			page := f.Page
			if page < 0 {
				page = 0
			}
			q = q.Limit(pageSize).Offset(page * pageSize)
		}

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

// getExpiredActive returns the active listings whose sale term has closed
// (ends_at IS NOT NULL AND ends_at < now) — auctions AND fixed sales alike
// (era-faithful fixed-sale expiry); only legacy NULL-ends_at rows never
// expire. The caller controls tenant scoping via the
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
		SetSerial(e.Serial).
		SetSellerAccountId(e.SellerAccountId).
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
		SetOfferWishSerial(e.OfferWishSerial).
		SetOfferWishOwnerId(e.OfferWishOwnerId).
		SetEndsAt(e.EndsAt).
		SetCurrentBid(e.CurrentBid).
		SetHighBidderId(e.HighBidderId).
		SetMinIncrement(e.MinIncrement).
		SetBidCount(e.BidCount).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt)
	return b.Build()
}
