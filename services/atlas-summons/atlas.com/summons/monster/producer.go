// Package monster declares atlas-summons' local view of the atlas-monsters
// command contract (COMMAND_TOPIC_MONSTER) and the providers used to credit a
// summon's owner with damage and to apply monster status effects.
//
// Services in this monorepo never import one another, so the envelope and body
// shapes here are re-declared to match the atlas-monsters consumer at
// services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go.
// The JSON tags MUST stay byte-identical to that consumer or owner credit /
// status application silently fail.
package monster

import (
	"atlas-summons/data/skill/effect"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

const (
	// EnvCommandTopic names the env var holding the atlas-monsters command topic.
	EnvCommandTopic = "COMMAND_TOPIC_MONSTER"

	// CommandTypeDamage credits a character with damage to a monster (XP / drops /
	// kill credit). Mirrors atlas-monsters CommandTypeDamage.
	CommandTypeDamage = "DAMAGE"
	// CommandTypeApplyStatus applies a monster status (stun/freeze). Mirrors
	// atlas-monsters CommandTypeApplyStatus.
	CommandTypeApplyStatus = "APPLY_STATUS"

	// CommandTypeAddPuppet registers a puppet in a field so the monster controller
	// picker biases toward the puppet's owner. Mirrors atlas-monsters
	// CommandTypeAddPuppet.
	CommandTypeAddPuppet = "ADD_PUPPET"
	// CommandTypeRemovePuppet clears a previously registered puppet. Mirrors
	// atlas-monsters CommandTypeRemovePuppet.
	CommandTypeRemovePuppet = "REMOVE_PUPPET"

	// sourceTypePlayerSkill classifies an APPLY_STATUS as originating from a
	// player skill, matching the value the channel service uses
	// (services/atlas-channel/.../monster/producer.go).
	sourceTypePlayerSkill = "PLAYER_SKILL"
)

// command is the atlas-monsters command envelope. Tags match
// monsters/kafka/consumer/monster/kafka.go:28-36.
type command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// damageCommandBody mirrors monsters/kafka/consumer/monster/kafka.go:43-47.
// Setting CharacterId to the summon owner is what credits the owner.
type damageCommandBody struct {
	CharacterId uint32   `json:"characterId"`
	Damages     []uint32 `json:"damages"`
	AttackType  byte     `json:"attackType"`
}

// applyStatusCommandBody mirrors monsters/kafka/consumer/monster/kafka.go:49-57.
type applyStatusCommandBody struct {
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          uint32           `json:"duration"`
	TickInterval      uint32           `json:"tickInterval"`
}

// addPuppetCommand is the FLAT (no Body envelope) ADD_PUPPET command. Tags MUST
// stay byte-identical to monsters/kafka/consumer/monster/kafka.go addPuppetCommand
// (worldId, channelId, mapId, instance, type, ownerCharacterId, x, y) or the
// monster controller bias silently fails to register.
type addPuppetCommand struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	Type             string     `json:"type"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	X                int16      `json:"x"`
	Y                int16      `json:"y"`
}

// removePuppetCommand is the FLAT (no Body envelope) REMOVE_PUPPET command. Tags
// MUST stay byte-identical to monsters/kafka/consumer/monster/kafka.go
// removePuppetCommand (worldId, channelId, mapId, instance, type,
// ownerCharacterId).
type removePuppetCommand struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	Type             string     `json:"type"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
}

// addPuppetProvider registers a puppet at (x,y) for ownerCharacterId in the field
// so atlas-monsters biases nearby monster controllers toward the owner (FR-4.x).
func addPuppetProvider(f field.Model, ownerCharacterId uint32, x int16, y int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(ownerCharacterId))
	value := &addPuppetCommand{
		WorldId:          f.WorldId(),
		ChannelId:        f.ChannelId(),
		MapId:            f.MapId(),
		Instance:         f.Instance(),
		Type:             CommandTypeAddPuppet,
		OwnerCharacterId: ownerCharacterId,
		X:                x,
		Y:                y,
	}
	return producer.SingleMessageProvider(key, value)
}

// AddPuppetProvider is the exported entry point for the summon processor.
func AddPuppetProvider(f field.Model, ownerCharacterId uint32, x int16, y int16) model.Provider[[]kafka.Message] {
	return addPuppetProvider(f, ownerCharacterId, x, y)
}

// removePuppetProvider clears a previously registered puppet for ownerCharacterId.
func removePuppetProvider(f field.Model, ownerCharacterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(ownerCharacterId))
	value := &removePuppetCommand{
		WorldId:          f.WorldId(),
		ChannelId:        f.ChannelId(),
		MapId:            f.MapId(),
		Instance:         f.Instance(),
		Type:             CommandTypeRemovePuppet,
		OwnerCharacterId: ownerCharacterId,
	}
	return producer.SingleMessageProvider(key, value)
}

// RemovePuppetProvider is the exported entry point for the summon processor.
func RemovePuppetProvider(f field.Model, ownerCharacterId uint32) model.Provider[[]kafka.Message] {
	return removePuppetProvider(f, ownerCharacterId)
}

// monsterDamageProvider credits ownerCharacterId with the supplied damage values
// against a monster (FR-4.2). CharacterId = owner ⇒ XP/drops/kill credit.
func monsterDamageProvider(f field.Model, monsterId uint32, ownerCharacterId uint32, damages []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &command[damageCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      CommandTypeDamage,
		Body: damageCommandBody{
			CharacterId: ownerCharacterId,
			Damages:     damages,
			AttackType:  0,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// MonsterDamageProvider is the exported entry point for the summon processor.
func MonsterDamageProvider(f field.Model, monsterId uint32, ownerCharacterId uint32, damages []uint32) model.Provider[[]kafka.Message] {
	return monsterDamageProvider(f, monsterId, ownerCharacterId, damages)
}

// monsterApplyStatusProvider applies the supplied statuses to a monster, sourced
// to the summon's owner and skill (FR-4.4). Duration comes from the skill effect.
func monsterApplyStatusProvider(f field.Model, monsterId uint32, ownerCharacterId uint32, skillId uint32, skillLevel byte, eff effect.Model, statuses map[string]int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	duration := eff.Duration()
	if duration < 0 {
		duration = 0
	}
	value := &command[applyStatusCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      CommandTypeApplyStatus,
		Body: applyStatusCommandBody{
			SourceType:        sourceTypePlayerSkill,
			SourceCharacterId: ownerCharacterId,
			SourceSkillId:     skillId,
			SourceSkillLevel:  uint32(skillLevel),
			Statuses:          statuses,
			Duration:          uint32(duration),
			TickInterval:      0,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// MonsterApplyStatusProvider is the exported entry point for the summon processor.
func MonsterApplyStatusProvider(f field.Model, monsterId uint32, ownerCharacterId uint32, skillId uint32, skillLevel byte, eff effect.Model, statuses map[string]int32) model.Provider[[]kafka.Message] {
	return monsterApplyStatusProvider(f, monsterId, ownerCharacterId, skillId, skillLevel, eff, statuses)
}
