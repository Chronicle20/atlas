// Package rps mirrors the atlas-rps Command envelope
// (services/atlas-rps/atlas.com/rps/kafka/message/rps/kafka.go) so
// atlas-channel can produce COMMAND_TOPIC_RPS messages without importing
// atlas-rps's package directly (service boundary). Keep the JSON shape
// (field names/tags) in sync with the atlas-rps side by hand - there is no
// shared library for this contract (Task 17b amendment).
package rps

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_RPS"

	// CommandTypeSelect submits the player's rock/paper/scissors throw for
	// the current round. CommandTypeContinue/Collect carry no body data of
	// their own.
	CommandTypeSelect   = "SELECT"
	CommandTypeContinue = "CONTINUE"
	CommandTypeCollect  = "COLLECT"
)

// Command represents a command sent to atlas-rps to act on a character's
// in-progress RPS session. E is the type-specific body payload.
type Command[E any] struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

// SelectCommandBody carries the player's rock/paper/scissors throw. The
// value is passed through RAW (0=Rock/1=Paper/2=Scissors) - unremapped, per
// docs/tasks/task-132-rps-npc-game/ida-rps-serverbound.md.
type SelectCommandBody struct {
	Throw byte `json:"throw"`
}

// ContinueCommandBody signals the player wants to play another round at the
// current rung. It carries no data.
type ContinueCommandBody struct {
}

// CollectCommandBody signals the player wants to end the session - banking
// the current prize if one is owed (StatusAwaitingDecision), or forfeiting
// otherwise. It carries no data.
type CollectCommandBody struct {
}
