package buff

import (
	"atlas-buffs/buff/stat"
	"github.com/google/uuid"
	"time"
)

type Model struct {
	id        uuid.UUID
	sourceId  uint32
	duration  int32
	changes   []stat.Model
	createdAt time.Time
	expiresAt time.Time
}

func (m Model) SourceId() uint32 {
	return m.sourceId
}

func (m Model) Expired() bool {
	return m.expiresAt.Before(time.Now())
}

func NewBuff(sourceId uint32, duration int32, changes []stat.Model) Model {
	return Model{
		id:        uuid.New(),
		sourceId:  sourceId,
		duration:  duration,
		changes:   changes,
		createdAt: time.Now(),
		expiresAt: time.Now().Add(time.Duration(duration) * time.Second),
	}
}
