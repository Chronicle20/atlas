// Package playernpc holds the atlas-messages producer side of
// COMMAND_TOPIC_PLAYER_NPC and the consumer side of
// EVENT_TOPIC_PLAYER_NPC_STATUS's COMMAND_SUCCEEDED/COMMAND_FAILED outcome
// events. It is NOT a field-for-field mirror: it only carries the subset of
// the shipped contract in
// services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/playernpc/kafka.go
// (Task 17/23a) that atlas-messages actually produces or consumes --
// DEPLOY/REMOVE (not REDEPLOY) on the command side, and
// COMMAND_SUCCEEDED/COMMAND_FAILED (not DEPLOYED/UPDATED/REMOVED/
// REPOSITIONED) on the status side. That upstream package is authoritative
// for every field name and JSON tag; this leaf message package carries no
// import of atlas-player-npcs.
package playernpc

import (
	"github.com/google/uuid"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic     topic.Token = "COMMAND_TOPIC_PLAYER_NPC"
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_PLAYER_NPC_STATUS"

	CommandTypeDeploy = "DEPLOY"
	CommandTypeRemove = "REMOVE"

	// EventTypeCommandSucceeded/Failed are the two StatusEvent.Type values
	// this service consumes on EnvEventTopicStatus (Task 23c). The other
	// values that share the topic -- DEPLOYED/UPDATED/REMOVED/REPOSITIONED
	// -- are domain events this service has no handler for and are ignored.
	EventTypeCommandSucceeded = "COMMAND_SUCCEEDED"
	EventTypeCommandFailed    = "COMMAND_FAILED"
)

// Command is the COMMAND_TOPIC_PLAYER_NPC envelope. Type selects Body's
// shape: DEPLOY carries CommandDeployBody, REMOVE carries
// CommandRemoveBody. TransactionId is left at uuid.Nil by both GM commands
// -- the GM path is not saga-driven -- and Requester carries the routing
// the outcome event is addressed back to (Task 23c); the consumer declines
// any outcome with a nil Requester as belonging to the saga path, not a GM.
type Command[E any] struct {
	CharacterId   uint32     `json:"characterId"`
	TransactionId uuid.UUID  `json:"transactionId"`
	Type          string     `json:"type"`
	Requester     *Requester `json:"requester,omitempty"`
	Body          E          `json:"body"`
}

// Requester is opaque routing set on a Command and echoed back unchanged
// on StatusCommandOutcomeBody: it identifies the invoking GM's character
// and current field so this service's status consumer can address the
// pink text without holding any local correlation state (atlas-messages
// runs a shared Kafka consumer group -- main.go's consumergroup.Resolve --
// so the pod that handles the outcome is not necessarily the pod that
// handled the command).
type Requester struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
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

// StatusEvent is the EVENT_TOPIC_PLAYER_NPC_STATUS envelope this service
// consumes. Type selects Body's shape -- COMMAND_SUCCEEDED/COMMAND_FAILED
// both carry StatusCommandOutcomeBody.
type StatusEvent[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

// StatusCommandOutcomeBody reports the outcome of one consumed
// COMMAND_TOPIC_PLAYER_NPC message back to whoever produced it. Code is
// empty on COMMAND_SUCCEEDED and one of the design §8.3 codes
// (pool_exhausted, map_full, duplicate, ineligible), unresolvable, or
// internal on COMMAND_FAILED. Requester is nil for the saga-driven path
// (atlas-npc-conversations' auto-deploy) -- this service's consumer
// declines any outcome with a nil Requester.
type StatusCommandOutcomeBody struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	CharacterId   uint32     `json:"characterId"`
	CommandType   string     `json:"commandType"`
	Code          string     `json:"code,omitempty"`
	Message       string     `json:"message,omitempty"`
	Requester     *Requester `json:"requester,omitempty"`
}
