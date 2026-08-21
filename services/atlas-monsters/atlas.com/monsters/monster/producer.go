package monster

import (
	mistKafka "atlas-monsters/kafka/message/mist"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// mistCreateCommandProvider builds a MIST_CREATE command keyed by the owning
// monster's unique id so concurrent commands from the same monster preserve
// their causal order on a single partition.
func mistCreateCommandProvider(t tenant.Model, body mistKafka.CreateCommandBody) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(body.OwnerId))
	value := &mistKafka.Command[mistKafka.CreateCommandBody]{
		Tenant: t.Id(),
		Type:   mistKafka.CommandTypeCreate,
		Body:   body,
	}
	return producer.SingleMessageProvider(key, value)
}

func createdStatusEventProvider(m Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusCreated, statusEventCreatedBody{ActorId: 0}, m.SpawnSourceType(), m.SpawnSourceId())
}

func destroyedStatusEventProvider(m Model, deathType byte) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusDestroyed, statusEventDestroyedBody{ActorId: 0, DeathType: deathType}, m.SpawnSourceType(), m.SpawnSourceId())
}

func statusEventProvider[E any](f field.Model, uniqueId uint32, monsterId uint32, theType string, body E, spawnSourceType string, spawnSourceId string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := statusEventFromField(f, uniqueId, monsterId, theType, body, spawnSourceType, spawnSourceId)
	return producer.SingleMessageProvider(key, &value)
}

func startControlStatusEventProvider(m Model) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusStartControl, statusEventStartControlBody{
		ActorId:            m.ControlCharacterId(),
		X:                  m.X(),
		Y:                  m.Y(),
		Stance:             m.Stance(),
		FH:                 m.Fh(),
		Team:               m.Team(),
		ControllerHasAggro: m.ControllerHasAggro(),
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

func aggroChangedStatusEventProvider(m Model, controllerCharacterId uint32, hasAggro bool) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusAggroChanged, statusEventAggroChangedBody{
		ControllerCharacterId: controllerCharacterId,
		ControllerHasAggro:    hasAggro,
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

func stopControlStatusEventProvider(m Model, characterId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusStopControl, statusEventStopControlBody{ActorId: characterId}, m.SpawnSourceType(), m.SpawnSourceId())
}

// damagedStatusEventProvider builds the DAMAGED event. damage is the amount
// this event applied (0 for a heal, which emits DAMAGED purely to refresh the
// HP bar); damageSummary is the monster's cumulative per-character totals.
func damagedStatusEventProvider(m Model, observerId uint32, actorId uint32, boss bool, damageSource string, damage uint32, damageSummary []entry) model.Provider[[]kafka.Message] {
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
		Damage:        damage,
		DamageSource:  damageSource,
		DamageEntries: damageEntries,
	}, m.SpawnSourceType(), m.SpawnSourceId())
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
		ReflectKind:       effect.ReflectKind(),
		ReflectPercent:    effect.ReflectPercent(),
		ReflectLtX:        effect.ReflectLtX(),
		ReflectLtY:        effect.ReflectLtY(),
		ReflectRbX:        effect.ReflectRbX(),
		ReflectRbY:        effect.ReflectRbY(),
		ReflectMaxDamage:  effect.ReflectMaxDamage(),
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

func statusEffectExpiredEventProvider(m Model, effect StatusEffect) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusEffectExpired, statusEffectExpiredBody{
		EffectId: effect.EffectId().String(),
		Statuses: effect.Statuses(),
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

func statusEffectCancelledEventProvider(m Model, effect StatusEffect) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusEffectCancelled, statusEffectCancelledBody{
		EffectId: effect.EffectId().String(),
		Statuses: effect.Statuses(),
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

func damageReflectedEventProvider(m Model, characterId uint32, reflectDamage uint32, reflectType string) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusDamageReflected, statusEventDamageReflectedBody{
		CharacterId:   characterId,
		ReflectDamage: reflectDamage,
		ReflectType:   reflectType,
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

// mpChangedStatusEventProvider builds a MP_CHANGED status event for any
// monster MP mutation that the channel must react to. Reason
// disambiguates the source (e.g., MP_EATER) so future passives can share
// the channel without expanding the consumer surface. Amount is the
// actual amount drained (post-clamp); MonsterMpAfter is the monster's
// MP after the deduction.
func mpChangedStatusEventProvider(m Model, characterId uint32, skillId uint32, reason string, amount uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusMpChanged, statusEventMpChangedBody{
		CharacterId:    characterId,
		SkillId:        skillId,
		Reason:         reason,
		Amount:         amount,
		MonsterMpAfter: m.Mp(),
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

func friendlyDropStatusEventProvider(m Model, itemCount uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusFriendlyDrop, statusEventFriendlyDropBody{ItemCount: itemCount}, m.SpawnSourceType(), m.SpawnSourceId())
}

func killedStatusEventProvider(m Model, killerId uint32, boss bool, damageSummary []entry, deathType byte) model.Provider[[]kafka.Message] {
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
		DeathType:     deathType,
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

// catchResolvedEventProvider keys on the character, not the map: the dedicated
// catch topic exists for atlas-consumables, whose ordering concern is
// per-character reservation handling.
func catchResolvedEventProvider(m Model, characterId uint32, itemId uint32, success bool, cause string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := statusEventFromField(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterCatchResolved, catchResolvedBody{
		CharacterId: characterId,
		ItemId:      itemId,
		Success:     success,
		Cause:       cause,
	}, m.SpawnSourceType(), m.SpawnSourceId())
	return producer.SingleMessageProvider(key, &value)
}

func caughtStatusEventProvider(m Model, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusCaught, statusEventCaughtBody{
		CharacterId: characterId,
		ItemId:      itemId,
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

func catchFailedStatusEventProvider(m Model, characterId uint32, itemId uint32, cause string) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusCatchFailed, statusEventCatchFailedBody{
		CharacterId: characterId,
		ItemId:      itemId,
		Cause:       cause,
	}, m.SpawnSourceType(), m.SpawnSourceId())
}

// nextSkillDecidedStatusEventProvider partitions on uniqueId so per-monster
// decision events stay ordered for atlas-channel's inbox writes.
func nextSkillDecidedStatusEventProvider(m Model, d nextSkillDecision) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.UniqueId()))
	value := statusEventFromField(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusNextSkillDecided, statusEventNextSkillDecidedBody{
		SkillId:                d.skillId,
		SkillLevel:             d.skillLevel,
		DecidedAtMs:            d.decidedAtMs,
		NextEligibleRepickAtMs: d.nextEligibleRepickAtMs,
	}, m.SpawnSourceType(), m.SpawnSourceId())
	return producer.SingleMessageProvider(key, &value)
}
