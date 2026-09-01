// Package playernpc holds the atlas-saga-orchestrator side of
// COMMAND_TOPIC_PLAYER_NPC and EVENT_TOPIC_PLAYER_NPC_STATUS. It mirrors
// the subset of the authoritative contract in
// services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/playernpc/kafka.go
// (Task 17, extended Task 23a) that this service produces or consumes: the
// DEPLOY command this service emits (with its TransactionId correlation
// field, but not Requester, which is the GM path's alone) and the
// COMMAND_SUCCEEDED/COMMAND_FAILED outcome events this service's consumer
// reads to complete a deploy_player_npc step. That consumer is
// authoritative, so this leaf message package carries no import of
// atlas-player-npcs.
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

	// EventTypeCommandSucceeded/Failed report the outcome of one consumed
	// COMMAND_TOPIC_PLAYER_NPC message back to whoever produced it (Task
	// 23a). DEPLOYED/UPDATED/REMOVED/REPOSITIONED share this topic too, but
	// this service only ever consumes these two outcome types.
	EventTypeCommandSucceeded = "COMMAND_SUCCEEDED"
	EventTypeCommandFailed    = "COMMAND_FAILED"
)

// Command is the COMMAND_TOPIC_PLAYER_NPC envelope. Type selects Body's
// shape; the saga-driven deploy_player_npc action only ever emits DEPLOY.
// TransactionId is what makes a command saga-driven -- a uuid.Nil one gets
// no reply.
type Command[E any] struct {
	CharacterId   uint32    `json:"characterId"`
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// CommandDeployBody is DEPLOY's body. deploy_player_npc (FR-6.2) always
// enforces eligibility -- the GM bypass path (design §9.2) belongs to
// atlas-messages, not the saga conversation operation.
type CommandDeployBody struct {
	WorldId            world.Id `json:"worldId"`
	MapId              _map.Id  `json:"mapId"`
	EnforceEligibility bool     `json:"enforceEligibility"`
}

// StatusEvent is the EVENT_TOPIC_PLAYER_NPC_STATUS envelope. This service
// only decodes it for COMMAND_SUCCEEDED/COMMAND_FAILED -- Type discriminates
// those from the DEPLOYED/UPDATED/REMOVED/REPOSITIONED domain events this
// service does not otherwise consume.
type StatusEvent[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

// StatusCommandOutcomeBody reports the outcome of one consumed
// COMMAND_TOPIC_PLAYER_NPC message (Task 23a). The correlation id lives on
// the body, not the envelope, because StatusEvent is shared with the
// pre-existing domain events. Code is empty on COMMAND_SUCCEEDED; on
// COMMAND_FAILED it carries the design §8.3 failure code (FR-6.3) --
// e.g. "pool_exhausted", "map_full", "ineligible".
type StatusCommandOutcomeBody struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	CommandType   string    `json:"commandType"`
	Code          string    `json:"code,omitempty"`
	Message       string    `json:"message,omitempty"`
}
