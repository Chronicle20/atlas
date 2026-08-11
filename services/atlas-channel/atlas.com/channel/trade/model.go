package trade

import (
	"github.com/google/uuid"
)

// Model is atlas-channel's read-only view of one live trade room, sourced from
// atlas-trades' GET /trades/rooms. Rooms are owned entirely by atlas-trades —
// atlas-channel never mutates one, it only asks whether a character occupies
// one (task-205 design §2.2), so the room's id is all this view carries.
type Model struct {
	id uuid.UUID
}

func (m Model) Id() uuid.UUID {
	return m.id
}
