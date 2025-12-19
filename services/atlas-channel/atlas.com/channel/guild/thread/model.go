package thread

import (
	"atlas-channel/guild/thread/reply"
	"github.com/google/uuid"
	"time"
)

type Model struct {
	tenantId   uuid.UUID
	guildId    uint32
	id         uint32
	posterId   uint32
	title      string
	message    string
	emoticonId uint32
	notice     bool
	createdAt  time.Time
	replies    []reply.Model
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

func (m Model) Title() string {
	return m.title
}

func (m Model) Message() string {
	return m.message
}

func (m Model) EmoticonId() uint32 {
	return m.emoticonId
}

func (m Model) Replies() []reply.Model {
	return m.replies
}

func (m Model) Notice() bool {
	return m.notice
}
