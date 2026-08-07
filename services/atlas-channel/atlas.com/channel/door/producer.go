package door

import (
	doormsg "atlas-channel/kafka/message/door"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func SpawnCommandProvider(f field.Model, ownerCharacterId, skillId uint32, level byte, x, y int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := doormsg.Command[doormsg.SpawnBody]{
		WorldId:          f.WorldId(),
		ChannelId:        f.ChannelId(),
		MapId:            f.MapId(),
		Instance:         f.Instance(),
		OwnerCharacterId: ownerCharacterId,
		Type:             doormsg.CommandTypeSpawn,
		Body: doormsg.SpawnBody{
			SkillId:    skillId,
			SkillLevel: level,
			X:          x,
			Y:          y,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RemoveCommandProvider(f field.Model, ownerCharacterId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := doormsg.Command[doormsg.RemoveBody]{
		WorldId:          f.WorldId(),
		ChannelId:        f.ChannelId(),
		MapId:            f.MapId(),
		Instance:         f.Instance(),
		OwnerCharacterId: ownerCharacterId,
		Type:             doormsg.CommandTypeRemove,
		Body: doormsg.RemoveBody{
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
