// Package rps mirrors the atlas-rps Command envelope
// (services/atlas-rps/atlas.com/rps/kafka/message/rps/kafka.go) so
// atlas-channel can produce COMMAND_TOPIC_RPS messages without importing
// atlas-rps's package directly (service boundary). Keep the JSON shape
// (field names/tags) in sync with the atlas-rps side by hand - there is no
// shared library for this contract (Task 17b amendment).
package rps

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_RPS"
	EnvEventTopic   = "EVENT_TOPIC_RPS"

	// CommandTypeSelect submits the player's rock/paper/scissors throw for
	// the current round. CommandTypeContinue/Collect carry no body data of
	// their own.
	CommandTypeSelect   = "SELECT"
	CommandTypeContinue = "CONTINUE"
	CommandTypeCollect  = "COLLECT"

	// EventTypeGameOpened/RoundResult/GameEnded mirror atlas-rps's Event.Type
	// values (services/atlas-rps/atlas.com/rps/kafka/message/rps/kafka.go).
	EventTypeGameOpened  = "GAME_OPENED"
	EventTypeRoundResult = "ROUND_RESULT"
	EventTypeGameEnded   = "GAME_ENDED"

	// Outcome values mirror atlas-rps's game.Outcome iota
	// (services/atlas-rps/atlas.com/rps/game/adjudicate.go): Lose=0, Tie=1,
	// Win=2. Defined locally rather than importing atlas-rps (service
	// boundary) - see docs/tasks/task-132-rps-npc-game for the mapping.
	OutcomeLose = 0
	OutcomeTie  = 1
	OutcomeWin  = 2

	// ReasonCollected/Lost/Quit/Disconnected are the terminal reasons a
	// GameEnded event can carry.
	ReasonCollected    = "collected"
	ReasonLost         = "lost"
	ReasonQuit         = "quit"
	ReasonDisconnected = "disconnected"
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

// Event represents an event emitted by atlas-rps as a session progresses. E
// is the type-specific body payload. Mirrors atlas-rps's Event[E] - keep the
// JSON shape in sync by hand (no shared library for this contract).
type Event[E any] struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

// GameOpenedEventBody signals a new RPS session has been opened for a
// character at an NPC. Ante is the participation fee / entry cost (in meso)
// charged to open the session; the channel's clientbound OPEN frame displays
// it.
type GameOpenedEventBody struct {
	NpcId uint32 `json:"npcId"`
	Ante  uint32 `json:"ante"`
}

// Prize describes a reward granted at a ladder rung.
type Prize struct {
	ItemId   item.Id `json:"itemId"`
	Quantity uint32  `json:"quantity"`
	Meso     uint32  `json:"meso"`
}

// RoundResultEventBody carries the outcome of a single adjudicated round.
type RoundResultEventBody struct {
	OpponentThrow byte  `json:"opponentThrow"`
	Outcome       int   `json:"outcome"`
	Rung          int   `json:"rung"`
	Prize         Prize `json:"prize"`
}

// GameEndedEventBody signals a session has terminated. GrantedPrize is only
// set when Reason is ReasonCollected - not consumed by the channel's END
// frame (bodyless), carried here only to keep the mirror complete.
type GameEndedEventBody struct {
	Reason       string `json:"reason"`
	GrantedPrize *Prize `json:"grantedPrize,omitempty"`
}
