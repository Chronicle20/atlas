// Package playernpc holds the atlas-saga-orchestrator producer side of
// COMMAND_TOPIC_PLAYER_NPC. It mirrors, field for field, the shipped
// consumer contract in
// services/atlas-player-npcs/atlas.com/player-npcs/kafka/message/playernpc/kafka.go
// (Task 17) and atlas-messages' own producer-side mirror
// (services/atlas-messages/atlas.com/messages/kafka/message/playernpc/kafka.go)
// -- that consumer is authoritative, so this leaf message package carries
// no import of atlas-player-npcs.
package playernpc

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_PLAYER_NPC"

	CommandTypeDeploy = "DEPLOY"
)

// Command is the COMMAND_TOPIC_PLAYER_NPC envelope. Type selects Body's
// shape; the saga-driven deploy_player_npc action only ever emits DEPLOY.
type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// CommandDeployBody is DEPLOY's body. deploy_player_npc (FR-6.2) always
// enforces eligibility -- the GM bypass path (design §9.2) belongs to
// atlas-messages, not the saga conversation operation.
type CommandDeployBody struct {
	WorldId            world.Id `json:"worldId"`
	MapId              _map.Id  `json:"mapId"`
	EnforceEligibility bool     `json:"enforceEligibility"`
}
