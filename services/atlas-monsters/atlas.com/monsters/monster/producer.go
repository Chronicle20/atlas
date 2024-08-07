package monster

import (
	"atlas-monsters/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func emitCreated(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32) model.Provider[[]kafka.Message] {
	return emitEvent(tenant, worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusCreated, statusEventCreatedBody{ActorId: 0})
}

func emitDestroyed(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32) model.Provider[[]kafka.Message] {
	return emitEvent(tenant, worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusDestroyed, statusEventDestroyedBody{ActorId: 0})
}

func emitEvent[E any](tenant tenant.Model, worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32, theType string, body E) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &statusEvent[E]{
		Tenant:    tenant,
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

func emitStartControl(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	return emitEvent(tenant, worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusStartControl, statusEventStartControlBody{ActorId: characterId})
}

func emitStopControl(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	return emitEvent(tenant, worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusStopControl, statusEventStopControlBody{ActorId: characterId})
}

func emitKilled(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32, x int16, y int16, killerId uint32, damageSummary []entry) model.Provider[[]kafka.Message] {
	var damageEntries []damageEntry
	for _, e := range damageSummary {
		damageEntries = append(damageEntries, damageEntry{
			CharacterId: e.CharacterId,
			Damage:      e.Damage,
		})
	}

	return emitEvent(tenant, worldId, channelId, mapId, uniqueId, monsterId, EventMonsterStatusKilled, statusEventKilledBody{
		X:             x,
		Y:             y,
		ActorId:       killerId,
		DamageEntries: damageEntries,
	})
}

func emitMove(tenant tenant.Model, worldId byte, channelId byte, uniqueId uint32, observerId uint32,
	skillPossible bool, skill int8, skillId int16, skillLevel int16, multiTarget []position,
	randTimes []int32, movement movement) model.Provider[[]kafka.Message] {

	key := producer.CreateKey(int(uniqueId))

	value := &movementEvent{
		Tenant:        tenant,
		WorldId:       worldId,
		ChannelId:     channelId,
		UniqueId:      uniqueId,
		ObserverId:    observerId,
		SkillPossible: skillPossible,
		Skill:         skill,
		SkillId:       skillId,
		SkillLevel:    skillLevel,
		MultiTarget:   multiTarget,
		RandomTimes:   randTimes,
		Movement:      movement,
	}
	return producer.SingleMessageProvider(key, value)
}
