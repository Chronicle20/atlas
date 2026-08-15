// Package event is the events service's OWN outbound contract: what should be
// rendered, not how. atlas-events never builds a packet (FR-B12) — it names the
// visual and the gameplay bytes, and atlas-channel maps them onto a writer it
// already has registered.
package event

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventTopicEventVisual = "EVENT_TOPIC_EVENT_VISUAL"
	VisualTypeShow           = "SHOW"
	VisualTypeHide           = "HIDE"

	// VisualContiMove is the enemy-ship visual. The name selects the writer on
	// the channel side; the state/subState bytes are gameplay content carried
	// in the body, so a future visual needs no new event type.
	VisualContiMove = "CONTI_MOVE"
)

type VisualEvent[E any] struct {
	OccurrenceId uuid.UUID  `json:"occurrenceId"`
	WorldId      world.Id   `json:"worldId"`
	ChannelId    channel.Id `json:"channelId"`
	MapId        _map.Id    `json:"mapId"`
	Type         string     `json:"type"`
	Body         E          `json:"body"`
}

type ShowVisualBody struct {
	Visual   string `json:"visual"`
	State    byte   `json:"state"`
	SubState byte   `json:"subState"`
	Bgm      string `json:"bgm,omitempty"`
}

type HideVisualBody struct {
	Visual   string `json:"visual"`
	State    byte   `json:"state"`
	SubState byte   `json:"subState"`
}
