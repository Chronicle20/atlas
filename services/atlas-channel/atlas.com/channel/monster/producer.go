package monster

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func damageCommandProvider(m _map.Model, monsterId uint32, characterId uint32, damage uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &command[damageCommandBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MonsterId: monsterId,
		Type:      CommandTypeDamage,
		Body: damageCommandBody{
			CharacterId: characterId,
			Damage:      damage,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
