package rps

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_RPS"
	EnvEventTopic   = "EVENT_TOPIC_RPS"

	// CommandTypeBegin opens the first round of an already-created (StatusOpen)
	// session: the player clicked "Start" on the board (serverbound RPS_ACTION
	// sub-op 0). CommandTypeSelect submits the player's rock/paper/scissors
	// throw for the current round. CommandTypeContinue/Collect/Quit carry no
	// body data of their own. StartGame (session creation) is not a Command
	// here; it arrives via REST (see Task 11).
	CommandTypeBegin    = "BEGIN"
	CommandTypeSelect   = "SELECT"
	CommandTypeContinue = "CONTINUE"
	CommandTypeCollect  = "COLLECT"
	CommandTypeQuit     = "QUIT"
	// CommandTypeRetry restarts a lost game: it re-charges the entry fee and
	// reopens a fresh round (the client's post-loss "Restart" button).
	CommandTypeRetry = "RETRY"

	EventTypeGameOpened   = "GAME_OPENED"
	EventTypeRoundStarted = "ROUND_STARTED"
	EventTypeRoundResult  = "ROUND_RESULT"
	EventTypeGameEnded    = "GAME_ENDED"

	// ReasonCollected/Quit/Disconnected are the terminal reasons a GameEnded
	// event can carry. GrantedPrize is only populated when Reason is
	// ReasonCollected. (A loss no longer emits GameEnded on its own — the loss
	// keeps the session for the player's Exit/Retry, which then emits
	// ReasonQuit; see game.Processor.Select's OutcomeLose branch.)
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

// BeginCommandBody signals the player clicked "Start" on the board to open the
// first round of an already-open session. It carries no data.
type BeginCommandBody struct {
}

// SelectCommandBody carries the player's rock/paper/scissors throw.
type SelectCommandBody struct {
	Throw byte `json:"throw"`
}

// ContinueCommandBody signals the player wants to play another round at the
// current rung. It carries no data.
type ContinueCommandBody struct {
}

// CollectCommandBody signals the player wants to bank their current prize
// and end the session. It carries no data.
type CollectCommandBody struct {
}

// QuitCommandBody signals the player is abandoning the session, forfeiting
// any unclaimed prize. It carries no data.
type QuitCommandBody struct {
}

// RetryCommandBody signals the player wants to restart after a loss (paying
// the entry fee again). It carries no data.
type RetryCommandBody struct {
}

// Event represents an event emitted by atlas-rps as a session progresses.
// E is the type-specific body payload.
type Event[E any] struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

// GameOpenedEventBody signals a new RPS session has been opened for a
// character at an NPC. Ante is the participation fee / entry cost (in meso)
// charged to open the session, sourced from the reward ladder's
// EntryCostMeso; the channel's clientbound OPEN frame displays it.
type GameOpenedEventBody struct {
	NpcId uint32 `json:"npcId"`
	Ante  uint32 `json:"ante"`
}

// RoundStartedEventBody signals a round is now open for the player's throw -
// the channel translates it to the clientbound START_SELECT frame (mode 9),
// which enables the client's R/P/S buttons and arms its selection timer. Rung
// is the rung the round is being played at (0-based; informational - the frame
// itself is bodyless). Emitted on the first round (BEGIN) and each subsequent
// round the player advances to (CONTINUE); NOT on a tie replay, which the
// client re-enables locally with no server frame.
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
// set when Reason is ReasonCollected.
type GameEndedEventBody struct {
	Reason       string `json:"reason"`
	GrantedPrize *Prize `json:"grantedPrize,omitempty"`
}
