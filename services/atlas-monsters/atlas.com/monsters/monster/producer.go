package monster

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdStatusEventProvider(worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusCreated, statusEventCreatedBody{ActorId: 0})
}

func destroyedStatusEventProvider(worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusDestroyed, statusEventDestroyedBody{ActorId: 0})
}

func statusEventProvider[E any](worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32, theType string, body E) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &statusEvent[E]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		UniqueId:  uniqueId,
		MonsterId: monsterId,
		Type:      theType,
		Body:      body,
	}
	return producer.SingleMessageProvider(key, value)
}

func startControlStatusEventProvider(m Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId(), EventMonsterStatusStartControl, statusEventStartControlBody{
		ActorId: m.ControlCharacterId(),
		X:       m.X(),
		Y:       m.Y(),
		Stance:  m.Stance(),
		FH:      m.Fh(),
		Team:    m.Team(),
	})
}

func stopControlStatusEventProvider(worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusStopControl, statusEventStopControlBody{ActorId: characterId})
}

func damagedStatusEventProvider(worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32, x int16, y int16, actorId uint32, damageSummary []entry) model.Provider[[]kafka.Message] {
	var damageEntries []damageEntry
	for _, e := range damageSummary {
		damageEntries = append(damageEntries, damageEntry{
			CharacterId: e.CharacterId,
			Damage:      e.Damage,
		})
	}

	return statusEventProvider(worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusDamaged, statusEventDamagedBody{
		X:             x,
		Y:             y,
		ActorId:       actorId,
		DamageEntries: damageEntries,
	})
}

func killedStatusEventProvider(worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32, x int16, y int16, killerId uint32, damageSummary []entry) model.Provider[[]kafka.Message] {
	var damageEntries []damageEntry
	for _, e := range damageSummary {
		damageEntries = append(damageEntries, damageEntry{
			CharacterId: e.CharacterId,
			Damage:      e.Damage,
		})
	}

	return statusEventProvider(worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusKilled, statusEventKilledBody{
		X:             x,
		Y:             y,
		ActorId:       killerId,
		DamageEntries: damageEntries,
	})
}
