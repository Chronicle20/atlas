package timer

import (
	characterKafka "atlas-maps/kafka/message/character"
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func mapTimerStartedProvider(transactionId uuid.UUID, f field.Model, characterId uint32, seconds uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mapKafka.StatusEvent[mapKafka.MapTimerStarted]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeMapTimerStarted,
		Body: mapKafka.MapTimerStarted{
			CharacterId: characterId,
			Seconds:     seconds,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMapProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &characterKafka.Command[characterKafka.ChangeMapBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          characterKafka.CommandChangeMap,
		Body: characterKafka.ChangeMapBody{
			ChannelId: channelId,
			MapId:     mapId,
			Instance:  uuid.Nil, // forced-return always targets non-instanced field
			PortalId:  0,        // default spawn portal
		},
	}
	return producer.SingleMessageProvider(key, value)
}
