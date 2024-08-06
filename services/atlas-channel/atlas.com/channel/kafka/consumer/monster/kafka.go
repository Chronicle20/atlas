package monster

import "atlas-channel/tenant"

const (
	EnvEventTopicMonsterStatus = "EVENT_TOPIC_MONSTER_STATUS"

	consumerNameStatus = "monster_status_event"

	EventMonsterStatusCreated      = "CREATED"
	EventMonsterStatusDestroyed    = "DESTROYED"
	EventMonsterStatusStartControl = "START_CONTROL"
	EventMonsterStatusStopControl  = "STOP_CONTROL"
	EventMonsterStatusKilled       = "KILLED"
)

type statusEvent[E any] struct {
	Tenant    tenant.Model `json:"tenant"`
	WorldId   byte         `json:"worldId"`
	ChannelId byte         `json:"channelId"`
	MapId     uint32       `json:"mapId"`
	UniqueId  uint32       `json:"uniqueId"`
	MonsterId uint32       `json:"monsterId"`
	Type      string       `json:"type"`
	Body      E            `json:"body"`
}

type statusEventCreatedBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventDestroyedBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventStartControlBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventStopControlBody struct {
	ActorId uint32 `json:"actorId"`
}

type statusEventKilledBody struct {
	X             int16         `json:"x"`
	Y             int16         `json:"y"`
	ActorId       uint32        `json:"actorId"`
	DamageEntries []damageEntry `json:"damageEntries"`
}

type damageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      int64  `json:"damage"`
}
