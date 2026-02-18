package monster

import (
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdStatusEventProvider(m Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusCreated, statusEventCreatedBody{ActorId: 0})
}

func destroyedStatusEventProvider(m Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusDestroyed, statusEventDestroyedBody{ActorId: 0})
}

func statusEventProvider[E any](f field.Model, uniqueId uint32, monsterId uint32, theType string, body E) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := statusEventFromField(f, uniqueId, monsterId, theType, body)
	return producer.SingleMessageProvider(key, &value)
}

func startControlStatusEventProvider(m Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusStartControl, statusEventStartControlBody{
		ActorId: m.ControlCharacterId(),
		X:       m.X(),
		Y:       m.Y(),
		Stance:  m.Stance(),
		FH:      m.Fh(),
		Team:    m.Team(),
	})
}

func stopControlStatusEventProvider(m Model, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusStopControl, statusEventStopControlBody{ActorId: characterId})
}

func damagedStatusEventProvider(m Model, observerId uint32, actorId uint32, boss bool, damageSummary []entry) model.Provider[[]kafka.Message] {
	var damageEntries []damageEntry
	for _, e := range damageSummary {
		damageEntries = append(damageEntries, damageEntry{
			CharacterId: e.CharacterId,
			Damage:      e.Damage,
		})
	}

	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusDamaged, statusEventDamagedBody{
		X:             m.X(),
		Y:             m.Y(),
		ObserverId:    observerId,
		ActorId:       actorId,
		Boss:          boss,
		DamageEntries: damageEntries,
	})
}

func statusEffectAppliedEventProvider(m Model, effect StatusEffect) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusEffectApplied, statusEffectAppliedBody{
		EffectId:          effect.EffectId().String(),
		SourceType:        effect.SourceType(),
		SourceCharacterId: effect.SourceCharacterId(),
		SourceSkillId:     effect.SourceSkillId(),
		SourceSkillLevel:  effect.SourceSkillLevel(),
		Statuses:          effect.Statuses(),
		Duration:          uint32(effect.Duration().Milliseconds()),
	})
}

func statusEffectExpiredEventProvider(m Model, effect StatusEffect) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusEffectExpired, statusEffectExpiredBody{
		EffectId: effect.EffectId().String(),
		Statuses: effect.Statuses(),
	})
}

func statusEffectCancelledEventProvider(m Model, effect StatusEffect) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusEffectCancelled, statusEffectCancelledBody{
		EffectId: effect.EffectId().String(),
		Statuses: effect.Statuses(),
	})
}

func damageReflectedEventProvider(m Model, characterId uint32, reflectDamage uint32, reflectType string) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusDamageReflected, statusEventDamageReflectedBody{
		CharacterId:   characterId,
		ReflectDamage: reflectDamage,
		ReflectType:   reflectType,
	})
}

func friendlyDropStatusEventProvider(f field.Model, uniqueId uint32, monsterId uint32, itemCount uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(f, uniqueId, monsterId, EventMonsterStatusFriendlyDrop, statusEventFriendlyDropBody{ItemCount: itemCount})
}

func killedStatusEventProvider(m Model, killerId uint32, boss bool, damageSummary []entry) model.Provider[[]kafka.Message] {
	var damageEntries []damageEntry
	for _, e := range damageSummary {
		damageEntries = append(damageEntries, damageEntry{
			CharacterId: e.CharacterId,
			Damage:      e.Damage,
		})
	}

	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusKilled, statusEventKilledBody{
		X:             m.X(),
		Y:             m.Y(),
		ActorId:       killerId,
		Boss:          boss,
		DamageEntries: damageEntries,
	})
}
