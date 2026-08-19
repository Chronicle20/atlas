// Package event mirrors atlas-events' EVENT_TOPIC_EVENT_VISUAL contract
// (services/atlas-events/atlas.com/events/kafka/message/event/kafka.go).
//
// atlas-events OWNS this contract; this file is a copy because the two
// services live in separate Go modules and nothing in the compiler links
// them. atlas-events never builds a packet (FR-B12) -- it names the visual
// and the gameplay bytes, and this consumer's whole job is to map that onto
// a writer atlas-channel already has registered.
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
	// the channel side, which resolves the client wire state/subState bytes
	// from the tenant's ContiMove writer options table (DOM-25) -- they are
	// not carried on this event. atlas-events names the visual and whether
	// it is being shown or hidden (Type); the wire bytes are per-tenant
	// config the channel side alone resolves.
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
	Visual string `json:"visual"`
	Bgm    string `json:"bgm,omitempty"`
}

type HideVisualBody struct {
	Visual string `json:"visual"`
}
