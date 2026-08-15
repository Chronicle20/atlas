package occurrence

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func getByIdProvider(id uuid.UUID) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		return database.Query[Entity](db, map[string]interface{}{"id": id})
	}
}

func getActiveByTypeProvider(theType string) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, map[string]interface{}{"type": theType, "state": StateActive})
	}
}

func getActiveByVoyageProvider(voyageId uuid.UUID, worldId world.Id, channelId channel.Id) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		return database.SliceQuery[Entity](db, map[string]interface{}{
			"voyage_id":  voyageId,
			"world_id":   uint8(worldId),
			"channel_id": uint8(channelId),
			"state":      StateActive,
		})
	}
}

// visualsInMapProvider answers the channel's generic "what visuals are active
// in this map" question (FR-API8). It does not filter on type — the channel
// asks a generic question, and filtering on a concrete event type here would
// be the generic layer naming that type.
func visualsInMapProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var results []Entity
		err := db.
			Joins("JOIN event_occurrence_map m ON m.occurrence_id = event_occurrence.id").
			Where("m.map_id = ? AND m.visual", uint32(mapId)).
			Where("event_occurrence.world_id = ? AND event_occurrence.channel_id = ?", uint8(worldId), uint8(channelId)).
			Where("event_occurrence.state = ?", StateActive).
			Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(results)
	}
}
