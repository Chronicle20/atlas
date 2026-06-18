package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const EnvEventTopicDoorStatus = "EVENT_TOPIC_DOOR_STATUS"

const (
	EventDoorStatusCreated     = "CREATED"
	EventDoorStatusRemoved     = "REMOVED"
	EventDoorStatusSlotChanged = "SLOT_CHANGED"
)

// Removal reasons (FR-6.1/6.2).
const (
	RemoveReasonExpiry         = "EXPIRY"
	RemoveReasonLogout         = "LOGOUT"
	RemoveReasonChannelChanged = "CHANNEL_CHANGED"
	RemoveReasonLeftField      = "LEFT_FIELD"
	RemoveReasonRecast         = "RECAST"
	RemoveReasonPartyLeft      = "PARTY_LEFT"
)

type StatusEvent[E any] struct {
	WorldId          world.Id     `json:"worldId"`
	ChannelId        channel.Id   `json:"channelId"`
	MapId            _map.Id      `json:"mapId"` // area field map (event key)
	Instance         uuid.UUID    `json:"instance"`
	PairId           uint32       `json:"pairId"`
	OwnerCharacterId character.Id `json:"ownerCharacterId"`
	PartyId          uint32       `json:"partyId"`
	// ForCharacterId targets a membership-change visibility delta at a single
	// character: 0 means broadcast to the door's eligible set (owner + current
	// party members); non-zero means deliver this packet ONLY to that character,
	// bypassing the eligibility filter (used to spawn party doors to a joiner and
	// remove them from a leaver, who is no longer in the eligible set).
	ForCharacterId uint32 `json:"forCharacterId"`
	Type           string `json:"type"`
	Body           E      `json:"body"`
}

type CreatedBody struct {
	AreaDoorId   uint32   `json:"areaDoorId"`
	TownDoorId   uint32   `json:"townDoorId"`
	TownMapId    _map.Id  `json:"townMapId"`
	Slot         byte     `json:"slot"`
	TownPortalId uint32   `json:"townPortalId"`
	AreaX        point.X  `json:"areaX"`
	AreaY        point.Y  `json:"areaY"`
	TownX        point.X  `json:"townX"`
	TownY        point.Y  `json:"townY"`
	SkillId      skill.Id `json:"skillId"`
	SkillLevel   byte     `json:"skillLevel"`
	ExpiresAt    int64    `json:"expiresAt"` // unix-milli
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
	TownX        point.X `json:"townX"`
	TownY        point.Y `json:"townY"`
	AreaX        point.X `json:"areaX"`
	AreaY        point.Y `json:"areaY"`
}
