package trade

import (
	"github.com/google/uuid"
)

// Model is atlas-channel's read-only view of one live trade room, sourced from
// atlas-trades' GET /trades/rooms. Rooms are owned entirely by atlas-trades —
// atlas-channel never mutates one, it only asks whether a character occupies
// one (task-205 design §2.2).
type Model struct {
	id       uuid.UUID
	roomType byte
	state    string
}

func (m Model) Id() uuid.UUID {
	return m.id
}

// RoomType is 3 (trade) or 6 (cash trade) — the mini-room type the room was
// opened with.
func (m Model) RoomType() byte {
	return m.roomType
}

// State is atlas-trades' lifecycle state (design §3.1), carried verbatim.
func (m Model) State() string {
	return m.state
}
