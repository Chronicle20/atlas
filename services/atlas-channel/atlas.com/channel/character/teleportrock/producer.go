package teleportrock

import (
	teleportrock2 "atlas-channel/kafka/message/teleportrock"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func addMapCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &teleportrock2.Command[teleportrock2.AddMapCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          teleportrock2.CommandAddMap,
		Body:          teleportrock2.AddMapCommandBody{MapId: mapId, Vip: vip},
	}
	return producer.SingleMessageProvider(key, value)
}

func removeMapCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &teleportrock2.Command[teleportrock2.RemoveMapCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          teleportrock2.CommandRemoveMap,
		Body:          teleportrock2.RemoveMapCommandBody{MapId: mapId, Vip: vip},
	}
	return producer.SingleMessageProvider(key, value)
}
