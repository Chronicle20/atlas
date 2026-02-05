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

func damagedStatusEventProvider(m Model, actorId uint32, damageSummary []entry) model.Provider[[]kafka.Message] {
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
		ActorId:       actorId,
		DamageEntries: damageEntries,
	})
}

func killedStatusEventProvider(m Model, killerId uint32, damageSummary []entry) model.Provider[[]kafka.Message] {
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
		DamageEntries: damageEntries,
	})
}
