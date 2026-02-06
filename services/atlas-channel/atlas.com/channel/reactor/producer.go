package reactor

import (
	reactor2 "atlas-channel/kafka/message/reactor"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func HitCommandProvider(f field.Model, reactorId uint32, characterId uint32, stance uint16, skillId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(reactorId))
	value := &reactor2.Command[reactor2.HitCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
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
