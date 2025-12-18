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

func (m Model) PosterId() uint32 {
	return m.posterId
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

func (m Model) Message() string {
	return m.message
}
