package buff

import (
	"atlas-buffs/buff/stat"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Model struct {
	id        uuid.UUID
	sourceId  int32
	level     byte
	duration  int32
	changes   []stat.Model
	createdAt time.Time
	expiresAt time.Time
}

func (m Model) SourceId() int32 {
	return m.sourceId
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) Expired() bool {
	return m.expiresAt.Before(time.Now())
}

func (m Model) Duration() int32 {
	return m.duration
}

func (m Model) Changes() []stat.Model {
	return m.changes
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

func (m Model) ExpiresAt() time.Time {
	return m.expiresAt
}

var (
	ErrInvalidDuration = errors.New("duration must be positive")
	ErrEmptyChanges    = errors.New("changes cannot be empty")
)

func NewBuff(sourceId int32, level byte, duration int32, changes []stat.Model) (Model, error) {
	if duration <= 0 {
		return Model{}, ErrInvalidDuration
	}
	if len(changes) == 0 {
		return Model{}, ErrEmptyChanges
	}
	return Model{
		id:        uuid.New(),
		sourceId:  sourceId,
		level:     level,
		duration:  duration,
		changes:   changes,
		createdAt: time.Now(),
		expiresAt: time.Now().Add(time.Duration(duration) * time.Second),
	}, nil
}
