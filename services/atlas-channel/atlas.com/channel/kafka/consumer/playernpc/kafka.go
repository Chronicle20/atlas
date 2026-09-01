// Package playernpc is atlas-channel's consumer for
// EVENT_TOPIC_PLAYER_NPC_STATUS (Task 19). The envelope types below are a
// channel-side copy of atlas-player-npcs' Task 17 contract
// (services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/playernpc/kafka.go),
// field-for-field byte-identical so the same JSON deserializes on the
// channel side -- mirrors kafka/consumer/door/kafka.go's own copy of the
// atlas-doors D1 contract; atlas-channel cannot import atlas-player-npcs
// (separate Go module).
package playernpc

import (
	"time"

	"github.com/google/uuid"
)

// EnvEventTopicStatus is the Player NPC status event topic env key.
// Byte-identical to atlas-player-npcs' EnvEventTopicStatus.
const EnvEventTopicStatus = "EVENT_TOPIC_PLAYER_NPC_STATUS"

const (
	EventTypeDeployed     = "DEPLOYED"
	EventTypeUpdated      = "UPDATED"
	EventTypeRemoved      = "REMOVED"
	EventTypeRepositioned = "REPOSITIONED"
)

// StatusEvent is the EVENT_TOPIC_PLAYER_NPC_STATUS envelope. Type selects
// Body's shape: DEPLOYED/UPDATED carry StatusModel, REMOVED carries
// StatusRemovedBody, REPOSITIONED carries StatusRepositionedBody.
type StatusEvent[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

// StatusEquipment is one frozen equipment slot on the wire.
type StatusEquipment struct {
	Slot   int16  `json:"slot"`
	ItemId uint32 `json:"itemId"`
}

// StatusModel is DEPLOYED/UPDATED's full resource payload -- the same
// attribute set as playernpc.RestModel (Task 18).
type StatusModel struct {
	Id             uuid.UUID         `json:"id"`
	CharacterId    uint32            `json:"characterId"`
	Name           string            `json:"name"`
	WorldId        byte              `json:"worldId"`
	MapId          uint32            `json:"mapId"`
	ScriptId       uint32            `json:"scriptId"`
	ObjectId       uint32            `json:"objectId"`
	Gender         byte              `json:"gender"`
	Skin           byte              `json:"skin"`
	Face           uint32            `json:"face"`
	Hair           uint32            `json:"hair"`
	JobId          uint16            `json:"jobId"`
	X              int16             `json:"x"`
	Cy             int16             `json:"cy"`
	Fh             uint16            `json:"fh"`
	Rx0            int16             `json:"rx0"`
	Rx1            int16             `json:"rx1"`
	Dir            byte              `json:"dir"`
	WorldRank      uint32            `json:"worldRank"`
	OverallRank    uint32            `json:"overallRank"`
	WorldJobRank   uint32            `json:"worldJobRank"`
	OverallJobRank uint32            `json:"overallJobRank"`
	Equipment      []StatusEquipment `json:"equipment"`
	DeployedAt     time.Time         `json:"deployedAt"`
}

// StatusRemovedBody is REMOVED's body.
type StatusRemovedBody struct {
	Id       uuid.UUID `json:"id"`
	ObjectId uint32    `json:"objectId"`
	MapId    uint32    `json:"mapId"`
	WorldId  byte      `json:"worldId"`
}

// StatusRepositionedNpc is one repositioned occupant within
// StatusRepositionedBody's list.
type StatusRepositionedNpc struct {
	Id       uuid.UUID `json:"id"`
	ObjectId uint32    `json:"objectId"`
	X        int16     `json:"x"`
	Cy       int16     `json:"cy"`
	Fh       uint16    `json:"fh"`
	Rx0      int16     `json:"rx0"`
	Rx1      int16     `json:"rx1"`
}

// StatusRepositionedBody is REPOSITIONED's body: one event carrying every
// occupant a re-organization moved.
type StatusRepositionedBody struct {
	WorldId byte                    `json:"worldId"`
	MapId   uint32                  `json:"mapId"`
	Npcs    []StatusRepositionedNpc `json:"npcs"`
}
