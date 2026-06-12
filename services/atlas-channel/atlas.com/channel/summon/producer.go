package summon

import (
	summon2 "atlas-channel/kafka/message/summon"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func SpawnCommandProvider(f field.Model, ownerCharacterId uint32, skillId uint32, level byte, x int16, y int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(ownerCharacterId))
	value := &summon2.Command[summon2.SpawnCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      summon2.CommandTypeSpawn,
		Body: summon2.SpawnCommandBody{
			OwnerCharacterId: ownerCharacterId,
			SkillId:          skillId,
			SkillLevel:       level,
			X:                x,
			Y:                y,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func MoveCommandProvider(f field.Model, summonId uint32, senderCharacterId uint32, x int16, y int16, stance byte, rawMovement []byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &summon2.Command[summon2.MoveCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		SummonId:  summonId,
		Type:      summon2.CommandTypeMove,
		Body: summon2.MoveCommandBody{
			SummonId:          summonId,
			SenderCharacterId: senderCharacterId,
			X:                 x,
			Y:                 y,
			Stance:            stance,
			RawMovement:       rawMovement,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AttackCommandProvider(f field.Model, summonId uint32, senderCharacterId uint32, direction byte, targets []summon2.AttackTargetEntry) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &summon2.Command[summon2.AttackCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		SummonId:  summonId,
		Type:      summon2.CommandTypeAttack,
		Body: summon2.AttackCommandBody{
			SummonId:          summonId,
			SenderCharacterId: senderCharacterId,
			Direction:         direction,
			Targets:           targets,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DamageCommandProvider(f field.Model, summonId uint32, senderCharacterId uint32, damage int32, monsterIdFrom uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &summon2.Command[summon2.DamageCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		SummonId:  summonId,
		Type:      summon2.CommandTypeDamage,
		Body: summon2.DamageCommandBody{
			SummonId:          summonId,
			SenderCharacterId: senderCharacterId,
			Damage:            damage,
			MonsterIdFrom:     monsterIdFrom,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
