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
	// the current round. CommandTypeContinue/Collect/Quit carry no body
	// data of their own. StartGame is not a Command here; it arrives via
	// REST (see Task 11).
	CommandTypeSelect   = "SELECT"
	CommandTypeContinue = "CONTINUE"
	CommandTypeCollect  = "COLLECT"
	CommandTypeQuit     = "QUIT"

	EventTypeGameOpened  = "GAME_OPENED"
	EventTypeRoundResult = "ROUND_RESULT"
	EventTypeGameEnded   = "GAME_ENDED"

	// ReasonCollected/Lost/Quit/Disconnected are the terminal reasons a
	// GameEnded event can carry. GrantedPrize is only populated when
	// Reason is ReasonCollected.
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
// character at an NPC.
type GameOpenedEventBody struct {
	NpcId uint32 `json:"npcId"`
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
