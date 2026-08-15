package crimsonbalrog

import (
	event "atlas-events/kafka/message/event"
	monster "atlas-events/kafka/message/monster"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// showVisualEventProvider announces the boat-attack visual for one map. Keyed
// on the map id so a SHOW and its later HIDE for the same map land on one
// partition and cannot be reordered.
func showVisualEventProvider(occurrenceId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, visual string, state byte, subState byte, bgm string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &event.VisualEvent[event.ShowVisualBody]{
		OccurrenceId: occurrenceId,
		WorldId:      worldId,
		ChannelId:    channelId,
		MapId:        mapId,
		Type:         event.VisualTypeShow,
		Body: event.ShowVisualBody{
			Visual:   visual,
			State:    state,
			SubState: subState,
			Bgm:      bgm,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// spawnFieldCommandProvider asks atlas-monsters to spawn one monster at pos,
// tagged with the occurrence's provenance (FR-B22) so DESTROY_BY_SOURCE
// (Task 27) can later clean up exactly what this occurrence spawned. Keyed on
// the map id, same as the visual, so a map's spawns land on one partition.
func spawnFieldCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, monsterId uint32, pos Position, occurrenceId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &monster.FieldCommand[monster.SpawnFieldCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      monster.CommandTypeSpawnField,
		Body: monster.SpawnFieldCommandBody{
			MonsterId:       monsterId,
			X:               pos.X,
			Y:               pos.Y,
			SpawnSourceType: "EVENT",
			SpawnSourceId:   occurrenceId.String(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
