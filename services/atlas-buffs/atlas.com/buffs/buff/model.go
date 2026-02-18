package buff

import (
	"atlas-buffs/buff/stat"
	"encoding/json"
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

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id        uuid.UUID    `json:"id"`
		SourceId  int32        `json:"sourceId"`
		Level     byte         `json:"level"`
		Duration  int32        `json:"duration"`
		Changes   []stat.Model `json:"changes"`
		CreatedAt time.Time    `json:"createdAt"`
		ExpiresAt time.Time    `json:"expiresAt"`
	}{
		Id:        m.id,
		SourceId:  m.sourceId,
		Level:     m.level,
		Duration:  m.duration,
		Changes:   m.changes,
		CreatedAt: m.createdAt,
		ExpiresAt: m.expiresAt,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id        uuid.UUID    `json:"id"`
		SourceId  int32        `json:"sourceId"`
		Level     byte         `json:"level"`
		Duration  int32        `json:"duration"`
		Changes   []stat.Model `json:"changes"`
		CreatedAt time.Time    `json:"createdAt"`
		ExpiresAt time.Time    `json:"expiresAt"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.id = aux.Id
	m.sourceId = aux.SourceId
	m.level = aux.Level
	m.duration = aux.Duration
	m.changes = aux.Changes
	m.createdAt = aux.CreatedAt
	m.expiresAt = aux.ExpiresAt
	return nil
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
