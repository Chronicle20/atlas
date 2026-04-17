package shop

import (
	"atlas-merchant/listing"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

func getAllOpen() database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("state IN (?, ?)", byte(Open), byte(Maintenance)).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

func getExpired() database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.Where("expires_at IS NOT NULL AND expires_at < NOW() AND state IN (?, ?)", byte(Open), byte(Maintenance)).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}

type listingSearchRow struct {
	listing.Entity
	ShopTitle     string     `gorm:"column:shop_title"`
	ShopWorldId   world.Id   `gorm:"column:shop_world_id"`
	ShopChannelId channel.Id `gorm:"column:shop_channel_id"`
	ShopMapId     uint32     `gorm:"column:shop_map_id"`
}

func searchListingsByItemId(itemId uint32) database.EntityProvider[[]ListingSearchResult] {
	return func(db *gorm.DB) model.Provider[[]ListingSearchResult] {
		var rows []listingSearchRow
		err := db.Table("listings").
			Select("listings.*, shops.title AS shop_title, shops.world_id AS shop_world_id, shops.channel_id AS shop_channel_id, shops.map_id AS shop_map_id").
			Joins("JOIN shops ON shops.id = listings.shop_id").
			Where("listings.item_id = ? AND shops.state IN (?, ?)", itemId, byte(Open), byte(Maintenance)).
			Order("listings.price_per_bundle ASC").
			Find(&rows).Error
		if err != nil {
			return model.ErrorProvider[[]ListingSearchResult](err)
		}

		results := make([]ListingSearchResult, 0, len(rows))
		for _, r := range rows {
			lm, err := listing.Make(r.Entity)
			if err != nil {
				return model.ErrorProvider[[]ListingSearchResult](err)
			}
			results = append(results, ListingSearchResult{
				Listing:   lm,
				ShopId:    r.Entity.ShopId,
				Title:     r.ShopTitle,
				WorldId:   r.ShopWorldId,
				ChannelId: r.ShopChannelId,
				MapId:     r.ShopMapId,
			})
		}
		return model.FixedProvider(results)
	}
}
