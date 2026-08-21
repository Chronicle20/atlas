// Package playernpc holds the Kafka wire types for
// COMMAND_TOPIC_PLAYER_NPC and EVENT_TOPIC_PLAYER_NPC_STATUS (Task 17).
// It is a leaf message package -- no import of atlas-player-npcs/playernpc
// -- so kafka/consumer/playernpc and playernpc/producer.go do the mapping
// to/from the domain model themselves.
package playernpc

import (
	"time"

	"github.com/google/uuid"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic     = "COMMAND_TOPIC_PLAYER_NPC"
	EnvEventTopicStatus = "EVENT_TOPIC_PLAYER_NPC_STATUS"

	CommandTypeDeploy   = "DEPLOY"
	CommandTypeRedeploy = "REDEPLOY"
	CommandTypeRemove   = "REMOVE"

	EventTypeDeployed     = "DEPLOYED"
	EventTypeUpdated      = "UPDATED"
	EventTypeRemoved      = "REMOVED"
	EventTypeRepositioned = "REPOSITIONED"
)

// Command is the COMMAND_TOPIC_PLAYER_NPC envelope. Type selects Body's
// shape: DEPLOY carries CommandDeployBody, REDEPLOY CommandRedeployBody,
// REMOVE CommandRemoveBody.
type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// CommandPosition is DEPLOY's optional explicit position (PRD §5's
// {x, y}); a nil Position on CommandDeployBody lets the positioner choose.
type CommandPosition struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

// CommandDeployBody is DEPLOY's body (plan.md Task 17 table).
type CommandDeployBody struct {
	WorldId            world.Id         `json:"worldId"`
	MapId              _map.Id          `json:"mapId"`
	Position           *CommandPosition `json:"position,omitempty"`
	EnforceEligibility bool             `json:"enforceEligibility"`
}

// CommandRedeployBody addresses the Player NPC to refresh by
// (characterId, worldId, mapId). Processor.Redeploy (Task 15) takes only
// the row's internal id -- a caller that only knows the character and map
// (e.g. the LEVEL_CHANGED consumer, or a future rank-refresh trigger) does
// not have that id, so the consumer resolves it via Processor.GetByMap
// before calling Redeploy. plan.md's Task 17 table lists only
// (characterId, mapId) for REDEPLOY; WorldId is added here because
// GetByMap requires it and Processor deliberately exposes no
// characterId+mapId-only lookup (that lookup, entitiesByCharacter, is
// unexported administrator.go plumbing Remove/RemoveById use internally).
type CommandRedeployBody struct {
	WorldId world.Id `json:"worldId"`
	MapId   _map.Id  `json:"mapId"`
}

// CommandRemoveBody's MapId is nil for "every map" (Processor.Remove's
// mapId *_map.Id, design's bulk-delete filter).
type CommandRemoveBody struct {
	MapId *_map.Id `json:"mapId,omitempty"`
}

// StatusEvent is the EVENT_TOPIC_PLAYER_NPC_STATUS envelope. Type selects
// Body's shape: DEPLOYED/UPDATED carry StatusModel (the full resource,
// design §7/§8.3), REMOVED carries StatusRemovedBody, REPOSITIONED
// carries StatusRepositionedBody.
type StatusEvent[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

// StatusEquipment is one frozen equipment slot on the wire -- mirrors
// playernpc.EquipmentRestModel (Task 16).
type StatusEquipment struct {
	Slot   int16  `json:"slot"`
	ItemId uint32 `json:"itemId"`
}

// StatusModel is DEPLOYED/UPDATED's full resource payload -- the same
// attribute set as playernpc.RestModel (Task 16), independently defined
// here so this leaf message package stays free of a playernpc/ import.
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

// StatusRemovedBody is REMOVED's body (plan.md Task 17 table).
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
// occupant a re-organization (design §5.4) moved.
type StatusRepositionedBody struct {
	WorldId byte                    `json:"worldId"`
	MapId   uint32                  `json:"mapId"`
	Npcs    []StatusRepositionedNpc `json:"npcs"`
}
