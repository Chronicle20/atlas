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

	// CommandTypeBegin opens the first round of an already-open session (the
	// player clicked "Start"; serverbound RPS_ACTION sub-op 0). CommandTypeSelect
	// submits the player's rock/paper/scissors throw for the current round.
	// CommandTypeContinue/Collect carry no body data of their own.
	CommandTypeBegin    = "BEGIN"
	CommandTypeSelect   = "SELECT"
	CommandTypeContinue = "CONTINUE"
	CommandTypeCollect  = "COLLECT"
	// CommandTypeRetry restarts a lost game (re-charges the entry fee): the
	// client's post-loss "Restart" button (serverbound RPS_ACTION sub-op 5).
	CommandTypeRetry = "RETRY"

	// EventTypeGameOpened/RoundStarted/RoundResult/GameEnded mirror atlas-rps's
	// Event.Type values (services/atlas-rps/atlas.com/rps/kafka/message/rps/kafka.go).
	EventTypeGameOpened   = "GAME_OPENED"
	EventTypeRoundStarted = "ROUND_STARTED"
	EventTypeRoundResult  = "ROUND_RESULT"
	EventTypeGameEnded    = "GAME_ENDED"

	// Outcome values mirror atlas-rps's game.Outcome iota
	// (services/atlas-rps/atlas.com/rps/game/adjudicate.go): Lose=0, Tie=1,
	// Win=2. Defined locally rather than importing atlas-rps (service
	// boundary) - see docs/tasks/task-132-rps-npc-game for the mapping.
	OutcomeLose = 0
	OutcomeTie  = 1
	OutcomeWin  = 2

	// ReasonCollected/Quit/Disconnected are the terminal reasons a GameEnded
	// event can carry (a loss no longer emits GameEnded on its own — see
	// atlas-rps game.Processor.Select). The channel's END frame is bodyless and
	// reason-agnostic; these mirror the atlas-rps contract for completeness.
	ReasonCollected    = "collected"
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

// BeginCommandBody signals the player clicked "Start" to open the first round
// of an already-open session. It carries no data.
type BeginCommandBody struct {
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

// RetryCommandBody signals the player wants to restart after a loss (paying
// the entry fee again). It carries no data.
type RetryCommandBody struct {
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

// RoundStartedEventBody signals a round is now open for the player's throw;
// the channel translates it to the clientbound START_SELECT frame (mode 9).
// Rung is informational (the frame itself is bodyless). Mirrors atlas-rps's
// RoundStartedEventBody.
type RoundStartedEventBody struct {
	Rung int `json:"rung"`
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
