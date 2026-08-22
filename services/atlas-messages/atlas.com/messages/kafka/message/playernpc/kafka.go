// Package playernpc holds the atlas-messages producer side of
// COMMAND_TOPIC_PLAYER_NPC. It mirrors, field for field, the shipped
// consumer contract in
// services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/playernpc/kafka.go
// (Task 17) -- that consumer is authoritative, so this leaf message
// package carries no import of atlas-player-npcs.
package playernpc

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_PLAYER_NPC"

	CommandTypeDeploy = "DEPLOY"
	CommandTypeRemove = "REMOVE"
)

// Command is the COMMAND_TOPIC_PLAYER_NPC envelope. Type selects Body's
// shape: DEPLOY carries CommandDeployBody, REMOVE carries
// CommandRemoveBody.
type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// CommandPosition is DEPLOY's optional explicit position; the GM command
// always supplies one (the invoking GM's current position).
type CommandPosition struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

// CommandDeployBody is DEPLOY's body. The GM command (design §9.2) always
// sets EnforceEligibility false -- it bypasses the level and auto-deploy
// checks but the downstream Deploy() still enforces script-id availability
// and the per-map duplicate rule (FR-8.1).
type CommandDeployBody struct {
	WorldId            world.Id         `json:"worldId"`
	MapId              _map.Id          `json:"mapId"`
	Position           *CommandPosition `json:"position,omitempty"`
	EnforceEligibility bool             `json:"enforceEligibility"`
}

// CommandRemoveBody's MapId is nil for "every map" (FR-8.2).
type CommandRemoveBody struct {
	MapId *_map.Id `json:"mapId,omitempty"`
}
