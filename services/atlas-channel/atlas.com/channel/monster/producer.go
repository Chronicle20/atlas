package monster

import (
	monster2 "atlas-channel/kafka/message/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func UseSkillCommandProvider(f field.Model, monsterId uint32, characterId uint32, skillId byte, skillLevel byte) model.Provider[[]kafka.Message] {
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

func CancelStatusCommandProvider(f field.Model, monsterId uint32, statusTypes []string, sourceCharacterId uint32, sourceSkillId uint32, sourceSkillClass string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.CancelStatusCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeCancelStatus,
		Body: monster2.CancelStatusCommandBody{
			StatusTypes:       statusTypes,
			SourceCharacterId: sourceCharacterId,
			SourceSkillId:     sourceSkillId,
			SourceSkillClass:  sourceSkillClass,
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

// DamageReflectedStatusEventProvider produces a StatusEvent[DAMAGE_REFLECTED]
// describing reflect damage that should be applied to the attacker. The
// existing atlas-channel monster status consumer (handleDamageReflected)
// reads the event and decrements the character's HP. Emitting the event
// from the attack handler keeps the reflect math out of atlas-monsters'
// hot path while reusing the established status-event channel.
func DamageReflectedStatusEventProvider(f field.Model, uniqueId uint32, monsterId uint32, characterId uint32, reflectDamage uint32, reflectType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(uniqueId))
	value := &monster2.StatusEvent[monster2.StatusEventDamageReflectedBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		UniqueId:  uniqueId,
		MonsterId: monsterId,
		Type:      monster2.EventStatusDamageReflected,
		Body: monster2.StatusEventDamageReflectedBody{
			CharacterId:   characterId,
			ReflectDamage: reflectDamage,
			ReflectType:   reflectType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DamageCommandProvider(f field.Model, monsterId uint32, characterId uint32, damages []uint32, attackType byte) model.Provider[[]kafka.Message] {
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
			Damages:     damages,
			AttackType:  attackType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
