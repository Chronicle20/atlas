package occurrence

import (
	"time"

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

// monsterCounts answers the SET-derived counts backing MonsterTally (design
// §9.5): total observed, and how many remain alive.
func monsterCounts(db *gorm.DB) func(occurrenceId uuid.UUID) (int, int, error) {
	return func(occurrenceId uuid.UUID) (int, int, error) {
		var total int64
		if err := db.Model(&MonsterEntity{}).Where("occurrence_id = ?", occurrenceId).Count(&total).Error; err != nil {
			return 0, 0, err
		}

		var alive int64
		if err := db.Model(&MonsterEntity{}).Where("occurrence_id = ? AND alive = ?", occurrenceId, true).Count(&alive).Error; err != nil {
			return 0, 0, err
		}

		return int(total), int(alive), nil
	}
}

// ListFilters narrows the GET /events/occurrences collection query (FR-API6).
// The zero value of every field means "no filter" for that dimension.
type ListFilters struct {
	DefinitionId  uuid.UUID
	Type          string
	State         string
	WorldId       *world.Id
	ChannelId     *channel.Id
	MapId         *_map.Id
	VoyageId      uuid.UUID
	StartedAtFrom *time.Time
	StartedAtTo   *time.Time
}

// listPagedProvider builds the WHERE-scoped, single-PK PagedQuery backing
// GET /events/occurrences. mapId is the one filter that requires a join
// against the FR-API8 child table.
func listPagedProvider(page model.Page, f ListFilters) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		scoped := db
		if f.DefinitionId != uuid.Nil {
			scoped = scoped.Where("event_definition_id = ?", f.DefinitionId)
		}
		if f.Type != "" {
			scoped = scoped.Where("type = ?", f.Type)
		}
		if f.State != "" {
			scoped = scoped.Where("state = ?", f.State)
		}
		if f.WorldId != nil {
			scoped = scoped.Where("world_id = ?", uint8(*f.WorldId))
		}
		if f.ChannelId != nil {
			scoped = scoped.Where("channel_id = ?", uint8(*f.ChannelId))
		}
		if f.VoyageId != uuid.Nil {
			scoped = scoped.Where("voyage_id = ?", f.VoyageId)
		}
		if f.StartedAtFrom != nil {
			scoped = scoped.Where("started_at >= ?", *f.StartedAtFrom)
		}
		if f.StartedAtTo != nil {
			scoped = scoped.Where("started_at <= ?", *f.StartedAtTo)
		}
		if f.MapId != nil {
			scoped = scoped.
				Joins("JOIN event_occurrence_map m ON m.occurrence_id = event_occurrence.id").
				Where("m.map_id = ?", uint32(*f.MapId))
		}
		return database.PagedQuery[Entity](scoped, page)
	}
}
