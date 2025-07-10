package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func DamageCommandProvider(m _map.Model, monsterId uint32, characterId uint32, damage uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.DamageCommandBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeDamage,
		Body: monster2.DamageCommandBody{
			CharacterId: characterId,
			Damage:      damage,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
