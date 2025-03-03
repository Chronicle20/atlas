package cashshop

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func characterEnterCashShopStatusEventProvider(actorId uint32, m _map.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &statusEvent[characterMovementBody]{
		WorldId: byte(m.WorldId()),
		Type:    EventCashShopStatusTypeCharacterEnter,
		Body: characterMovementBody{
			CharacterId: actorId,
			ChannelId:   byte(m.ChannelId()),
			MapId:       uint32(m.MapId()),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func characterExitCashShopStatusEventProvider(actorId uint32, m _map.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &statusEvent[characterMovementBody]{
		WorldId: byte(m.WorldId()),
		Type:    EventCashShopStatusTypeCharacterExit,
		Body: characterMovementBody{
			CharacterId: actorId,
			ChannelId:   byte(m.ChannelId()),
			MapId:       uint32(m.MapId()),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
