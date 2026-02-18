package monster

import (
	monster2 "atlas-channel/kafka/message/monster"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func ApplyStatusCommandProvider(f field.Model, monsterId uint32, sourceCharacterId uint32, sourceSkillId uint32, sourceSkillLevel uint32, statuses map[string]int32, duration uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.ApplyStatusCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeApplyStatus,
		Body: monster2.ApplyStatusCommandBody{
			SourceType:        "PLAYER_SKILL",
			SourceCharacterId: sourceCharacterId,
			SourceSkillId:     sourceSkillId,
			SourceSkillLevel:  sourceSkillLevel,
			Statuses:          statuses,
			Duration:          duration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func UseSkillCommandProvider(f field.Model, monsterId uint32, characterId uint32, skillId uint16, skillLevel uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.UseSkillCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeUseSkill,
		Body: monster2.UseSkillCommandBody{
			CharacterId: characterId,
			SkillId:     skillId,
			SkillLevel:  skillLevel,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelStatusCommandProvider(f field.Model, monsterId uint32, statusTypes []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.CancelStatusCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeCancelStatus,
		Body: monster2.CancelStatusCommandBody{
			StatusTypes: statusTypes,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DamageFriendlyCommandProvider(f field.Model, attackedUniqueId uint32, observerUniqueId uint32, attackerUniqueId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(attackedUniqueId))
	value := &monster2.Command[monster2.DamageFriendlyCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: attackedUniqueId,
		Type:      monster2.CommandTypeDamageFriendly,
		Body: monster2.DamageFriendlyCommandBody{
			AttackerUniqueId: attackerUniqueId,
			ObserverUniqueId: observerUniqueId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DamageCommandProvider(f field.Model, monsterId uint32, characterId uint32, damage uint32, attackType byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.DamageCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeDamage,
		Body: monster2.DamageCommandBody{
			CharacterId: characterId,
			Damage:      damage,
			AttackType:  attackType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
