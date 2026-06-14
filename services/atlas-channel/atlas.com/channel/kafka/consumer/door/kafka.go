package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// EnvEventTopicDoorStatus is the door status event topic env key. Byte-identical
// to the atlas-doors D1 contract (services/atlas-doors/.../door/kafka.go).
const EnvEventTopicDoorStatus = "EVENT_TOPIC_DOOR_STATUS"

const (
	EventDoorStatusCreated     = "CREATED"
	EventDoorStatusRemoved     = "REMOVED"
	EventDoorStatusSlotChanged = "SLOT_CHANGED"
)

// Removal reasons (FR-6.1/6.2). Mirrors atlas-doors D1.
const (
	RemoveReasonExpiry         = "EXPIRY"
	RemoveReasonLogout         = "LOGOUT"
	RemoveReasonChannelChanged = "CHANNEL_CHANGED"
	RemoveReasonLeftField      = "LEFT_FIELD"
	RemoveReasonRecast         = "RECAST"
)

// StatusEvent is the channel-side copy of the atlas-doors D1 door status
// envelope. Field-for-field byte-identical to atlas-doors so the same JSON
// deserializes on the channel side.
type StatusEvent[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"` // area field map (event key)
	Instance         uuid.UUID  `json:"instance"`
	PairId           uint32     `json:"pairId"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	PartyId          uint32     `json:"partyId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

type CreatedBody struct {
	AreaDoorId   uint32  `json:"areaDoorId"`
	TownDoorId   uint32  `json:"townDoorId"`
	TownMapId    _map.Id `json:"townMapId"`
	Slot         byte    `json:"slot"`
	TownPortalId uint32  `json:"townPortalId"`
	AreaX        int16   `json:"areaX"`
	AreaY        int16   `json:"areaY"`
	TownX        int16   `json:"townX"`
	TownY        int16   `json:"townY"`
	SkillId      uint32  `json:"skillId"`
	SkillLevel   byte    `json:"skillLevel"`
	ExpiresAt    int64   `json:"expiresAt"` // unix-milli
}

type RemovedBody struct {
	AreaDoorId uint32  `json:"areaDoorId"`
	TownDoorId uint32  `json:"townDoorId"`
	TownMapId  _map.Id `json:"townMapId"`
	Slot       byte    `json:"slot"`
	Reason     string  `json:"reason"`
}

type SlotChangedBody struct {
	AreaDoorId   uint32  `json:"areaDoorId"`
	TownDoorId   uint32  `json:"townDoorId"`
	TownMapId    _map.Id `json:"townMapId"`
	OldSlot      byte    `json:"oldSlot"`
	NewSlot      byte    `json:"newSlot"`
	TownPortalId uint32  `json:"townPortalId"`
	TownX        int16   `json:"townX"`
	TownY        int16   `json:"townY"`
}
