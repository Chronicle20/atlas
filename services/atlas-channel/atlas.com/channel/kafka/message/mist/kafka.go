// Package mist mirrors atlas-maps' COMMAND_TOPIC_MIST / EVENT_TOPIC_MIST
// contract (services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go).
//
// atlas-maps OWNS this contract; this file is a copy because the two services
// live in separate Go modules and nothing in the compiler links them. Do not
// edit this file directly: change the owner, re-copy, and restore this header.
// tools/mist-contract-mirror-guard.sh fails CI on any other difference.
package mist

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_MIST"
	EnvEventTopic   = "EVENT_TOPIC_MIST"

	CommandTypeCreate = "CREATE"
	CommandTypeCancel = "CANCEL"

	EventTypeCreated   = "MIST_CREATED"
	EventTypeDestroyed = "MIST_DESTROYED"

	ReasonExpired   = "EXPIRED"
	ReasonCancelled = "CANCELLED"

	// TargetKind selects who a mist's per-tick effect is applied to. An empty
	// value means CHARACTER, so producers written before task-200 (the
	// atlas-monsters AREA_POISON path) keep working unchanged.
	TargetKindCharacter = "CHARACTER"
	TargetKindMonster   = "MONSTER"

	// EffectKind selects what the mist's per-tick effect does. An empty value
	// means DISEASE. DISEASE applies a named character status via
	// COMMAND_TOPIC_CHARACTER_BUFF; DAMAGE_OVER_TIME applies a damage-bearing
	// monster status via COMMAND_TOPIC_MONSTER APPLY_STATUS; RECOVERY restores
	// MP to the party members inside via COMMAND_TOPIC_CHARACTER CHANGE_MP;
	// PROTECTION shields the owner's party from damage and is evaluated in
	// atlas-channel on the damage path -- it has no atlas-maps tick at all.
	EffectKindDisease        = "DISEASE"
	EffectKindDamageOverTime = "DAMAGE_OVER_TIME"
	EffectKindProtection     = "PROTECTION"
	EffectKindRecovery       = "RECOVERY"
)

// Command is the envelope for mist commands published to EnvCommandTopic.
type Command[E any] struct {
	Tenant uuid.UUID `json:"tenant"`
	Type   string    `json:"type"`
	Body   E         `json:"body"`
}

// CreateCommandBody requests creation of a new mist on the named field.
type CreateCommandBody struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerType        string     `json:"ownerType"`
	OwnerId          uint32     `json:"ownerId"`
	OriginX          int16      `json:"originX"`
	OriginY          int16      `json:"originY"`
	LtX              int16      `json:"ltX"`
	LtY              int16      `json:"ltY"`
	RbX              int16      `json:"rbX"`
	RbY              int16      `json:"rbY"`
	Disease          string     `json:"disease"`
	DiseaseValue     int32      `json:"diseaseValue"`
	DiseaseDuration  int64      `json:"diseaseDuration"`
	Duration         int64      `json:"duration"`
	TickIntervalMs   int64      `json:"tickIntervalMs"`
	SourceSkillId    uint32     `json:"sourceSkillId"`
	SourceSkillLevel uint32     `json:"sourceSkillLevel"`
	// TargetKind is "CHARACTER" or "MONSTER"; empty means CHARACTER.
	TargetKind string `json:"targetKind"`
	// EffectKind is "DISEASE" or "DAMAGE_OVER_TIME"; empty means DISEASE.
	EffectKind string `json:"effectKind"`
	// RecoveryMp is the per-tick MP restored by a RECOVERY mist. Unlike
	// DiseaseValue -- which is target-derived and overwritten downstream by
	// atlas-monsters -- this magnitude is caster-derived and authoritative,
	// so it gets its own field rather than overloading DiseaseValue.
	RecoveryMp int32 `json:"recoveryMp"`
	// PartyMemberIds scopes a RECOVERY mist to the caster's party, snapshot
	// at cast time by the atlas-channel handler (atlas-maps has no party
	// client). Always includes the caster. Ignored by every other kind.
	PartyMemberIds []uint32 `json:"partyMemberIds"`
}

// CancelCommandBody requests cancellation of an existing mist by id.
type CancelCommandBody struct {
	MistId uuid.UUID `json:"mistId"`
}

// Event is the envelope for mist events published to EnvEventTopic.
type Event[E any] struct {
	Tenant    uuid.UUID  `json:"tenant"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MistId    uuid.UUID  `json:"mistId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// CreatedBody describes a mist that was just created.
type CreatedBody struct {
	OwnerType        string `json:"ownerType"`
	OwnerId          uint32 `json:"ownerId"`
	SourceSkillId    uint32 `json:"sourceSkillId"`
	SourceSkillLevel uint32 `json:"sourceSkillLevel"`
	Type             int32  `json:"type"`
	OriginX          int16  `json:"originX"`
	OriginY          int16  `json:"originY"`
	LtX              int16  `json:"ltX"`
	LtY              int16  `json:"ltY"`
	RbX              int16  `json:"rbX"`
	RbY              int16  `json:"rbY"`
	Duration         int64  `json:"duration"`
	// ElemAttr is the client's `nElemAttr`; SkillDelay is its `skillDelay`
	// draw delay (units of 100 ms). The existing `Type` field IS the client's
	// `nType` -- do not add a second key for it.
	ElemAttr   int32 `json:"elemAttr"`
	SkillDelay int16 `json:"skillDelay"`
	// EffectKind lets atlas-channel recognise a PROTECTION mist without
	// inferring it from the client-facing `Type` (nType) value. Carrying the
	// domain concept keeps nType a pure render detail -- see
	// mist.AffectedAreaTypeFor's doc comment.
	EffectKind string `json:"effectKind"`
}

// DestroyedBody describes a mist that was just destroyed.
type DestroyedBody struct {
	Reason string `json:"reason"`
}
