package buff

import (
	"atlas-channel/character/buff/stat"
	"time"
)

type Model struct {
	sourceId  int32
	duration  int32
	changes   []stat.Model
	createdAt time.Time
	expiresAt time.Time
}

func (m Model) SourceId() int32 {
	return m.sourceId
}

func (m Model) Changes() []stat.Model {
	return m.changes
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

func (m Model) Expired() bool {
	return m.expiresAt.Before(time.Now())
}

func (m Model) ExpiresAt() time.Time {
	return m.expiresAt
}

func NewBuff(sourceId int32, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time) Model {
	return Model{
		sourceId:  sourceId,
		duration:  duration,
		changes:   changes,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}
}
