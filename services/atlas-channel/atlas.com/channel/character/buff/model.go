package buff

import (
	"atlas-channel/character/buff/stat"
	"time"
)

type Model struct {
	sourceId  uint32
	duration  int32
	changes   []stat.Model
	createdAt time.Time
	expiresAt time.Time
}

func (m Model) SourceId() uint32 {
	return m.sourceId
}

func (m Model) Changes() []stat.Model {
	return m.changes
}

func (m Model) Duration() int32 {
	return m.duration
}

func NewBuff(sourceId uint32, duration int32, changes []stat.Model) Model {
	return Model{
		sourceId: sourceId,
		duration: duration,
		changes:  changes,
	}
}
