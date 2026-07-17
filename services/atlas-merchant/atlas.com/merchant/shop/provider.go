package shop

import (
	"time"

	"atlas-merchant/listing"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func getById(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("id = ?", id).First(&result).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrorProvider[Entity](ErrNotFound)
			}
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// getByCharacterId retrieves every shop for a character, unpaged. Kept for
// internal callers that genuinely need the complete (naturally small, at
// most one shop per ShopType) set: the character-logout consumer
// (kafka/consumer/character/consumer.go) that closes open CharacterShops on
// disconnect. The HTTP list route uses the paged sibling below.
func getByCharacterId(characterId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("character_id = ?", characterId).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

// getByCharacterIdPaged is the paged sibling of getByCharacterId, used by
// the GET /characters/{characterId}/merchants list route (task-117).
func getByCharacterIdPaged(characterId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("character_id = ?", characterId), page)
	}
}

func getActiveByCharacterIdAndType(characterId uint32, shopType ShopType) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("character_id = ? AND shop_type = ? AND state != ?", characterId, byte(shopType), byte(Closed)).First(&result).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrorProvider[Entity](ErrNotFound)
			}
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// getOwnerOccupiedShop resolves, authoritatively from the DB, the shop a
// character owns and is currently occupying: a personal shop in any non-Closed
// state, or a hired merchant in Draft/Maintenance (setup/management). An Open
// hired merchant is owner-detached and is deliberately NOT returned — the owner
// must re-enter maintenance to manage it. This backs GetShopForCharacter's
// fallback when the Redis occupancy cache is missing or stale, so an owner is
// never stranded from acting on their own shop.
func getOwnerOccupiedShop(characterId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where(
			"character_id = ? AND ((shop_type = ? AND state != ?) OR (shop_type = ? AND state IN (?, ?)))",
			characterId,
			byte(CharacterShop), byte(Closed),
			byte(HiredMerchant), byte(Draft), byte(Maintenance),
		).Order("created_at DESC").First(&result).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrorProvider[Entity](ErrNotFound)
			}
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(result)
	}
}

// getByField retrieves every non-closed shop on a field, unpaged. Kept for
// the internal CreateShop shop-proximity validation (processor.go), which
// needs the complete set to check placement distance against every existing
// shop, not one page of it. The HTTP list route uses the paged sibling below.
func getByField(worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("world_id = ? AND channel_id = ? AND map_id = ? AND instance_id = ? AND state != ?", worldId, channelId, mapId, instanceId, byte(Closed)).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

// getByFieldPaged is the paged sibling of getByField, used by the
// GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/merchants
// list route (task-117).
func getByFieldPaged(worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("world_id = ? AND channel_id = ? AND map_id = ? AND instance_id = ? AND state != ?", worldId, channelId, mapId, instanceId, byte(Closed)), page)
	}
}

// getAllOpenPaged is the full-table (bare GET /merchants) provider,
// paginated (task-117). The prior unpaged getAllOpen had no internal caller
// (only the now-converted handler used it), so it is deleted rather than
// kept alongside a paged sibling, per the Group A "delete, don't shadow"
// convention.
func getAllOpenPaged(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("state IN (?, ?)", byte(Open), byte(Maintenance)), page)
	}
}

// getExpired matches shops past their expires_at in every live state —
// including Draft, so a hired merchant abandoned during setup (never opened)
// is still reaped instead of blocking the character's shop slot forever.
// The cutoff is bound Go-side (portable across postgres/sqlite).
func getExpired() database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("expires_at IS NOT NULL AND expires_at < ? AND state IN (?, ?, ?)", time.Now(), byte(Draft), byte(Open), byte(Maintenance)).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

type listingSearchRow struct {
	listing.Entity
	ShopTitle       string     `gorm:"column:shop_title"`
	ShopWorldId     world.Id   `gorm:"column:shop_world_id"`
	ShopChannelId   channel.Id `gorm:"column:shop_channel_id"`
	ShopMapId       uint32     `gorm:"column:shop_map_id"`
	ShopCharacterId uint32     `gorm:"column:shop_character_id"`
	ShopShopType    byte       `gorm:"column:shop_shop_type"`
	ShopState       byte       `gorm:"column:shop_state"`
}

// searchListingsByItemIdPaged joins listings to shops for the shop-scanner
// search, paged (task-117). The .Table() form bypasses the automatic tenant
// callback (no bound schema), so tenant_id predicates are explicit here —
// which also keeps the COUNT tenant-scoped without a schema-bearing Model.
//
// This does NOT delegate to database.PagedQuery: that helper appends an
// unqualified `ORDER BY id` primary-key tiebreaker derived from the target
// entity's schema, which is ambiguous once this query JOINs shops (both
// "listings" and "shops" have an "id" column) and would fail against
// Postgres ("column reference \"id\" is ambiguous"). Pagination is
// hand-rolled here with an explicitly qualified `listings.id` tiebreaker
// instead, keeping the existing Where/Join filters intact.
func searchListingsByItemIdPaged(tenantId uuid.UUID, criteria ListingSearchCriteria, page model.Page) database.EntityProvider[model.Paged[ListingSearchResult]] {
	return func(db *gorm.DB) model.Provider[model.Paged[ListingSearchResult]] {
		if page.Number < 1 || page.Size < 1 {
			return model.ErrorProvider[model.Paged[ListingSearchResult]](fmt.Errorf("invalid page number=%d size=%d", page.Number, page.Size))
		}

		order := "listings.price_per_bundle ASC"
		if criteria.Descending {
			order = "listings.price_per_bundle DESC"
		}

		base := db.Table("listings").
			Joins("JOIN shops ON shops.id = listings.shop_id").
			Where("listings.item_id = ? AND listings.tenant_id = ? AND shops.tenant_id = ? AND shops.state IN (?, ?)", criteria.ItemId, tenantId, tenantId, byte(Open), byte(Maintenance))
		if criteria.WorldId != nil {
			base = base.Where("shops.world_id = ?", *criteria.WorldId)
		}

		var total int64
		if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
			return model.ErrorProvider[model.Paged[ListingSearchResult]](err)
		}

		var rows []listingSearchRow
		err := base.Session(&gorm.Session{}).
			Select("listings.*, shops.title AS shop_title, shops.world_id AS shop_world_id, shops.channel_id AS shop_channel_id, shops.map_id AS shop_map_id, shops.character_id AS shop_character_id, shops.shop_type AS shop_shop_type, shops.state AS shop_state").
			Order(order).
			Order("listings.id ASC").
			Offset((page.Number - 1) * page.Size).
			Limit(page.Size).
			Find(&rows).Error
		if err != nil {
			return model.ErrorProvider[model.Paged[ListingSearchResult]](err)
		}

		results := make([]ListingSearchResult, 0, len(rows))
		for _, r := range rows {
			lm, err := listing.Make(r.Entity)
			if err != nil {
				return model.ErrorProvider[model.Paged[ListingSearchResult]](err)
			}
			results = append(results, ListingSearchResult{
				Listing:     lm,
				ShopId:      r.Entity.ShopId,
				Title:       r.ShopTitle,
				WorldId:     r.ShopWorldId,
				ChannelId:   r.ShopChannelId,
				MapId:       r.ShopMapId,
				ShopOwnerId: r.ShopCharacterId,
				ShopType:    ShopType(r.ShopShopType),
				State:       State(r.ShopState),
			})
		}
		return model.FixedProvider(model.Paged[ListingSearchResult]{Items: results, Total: int(total), Page: page})
	}
}
