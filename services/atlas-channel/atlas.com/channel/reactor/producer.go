package reactor

import (
	reactor2 "atlas-channel/kafka/message/reactor"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func HitCommandProvider(m _map.Model, reactorId uint32, characterId uint32, stance uint16, skillId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(reactorId))
	value := &reactor2.Command[reactor2.HitCommandBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		Type:      reactor2.CommandTypeHit,
		Body: reactor2.HitCommandBody{
			ReactorId:   reactorId,
			CharacterId: characterId,
			Stance:      stance,
			SkillId:     skillId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
