package reply

import "time"

type Model struct {
	id        uint32
	posterId  uint32
	message   string
	createdAt time.Time
}

func (m Model) Id() uint32 {
	return m.id
}
