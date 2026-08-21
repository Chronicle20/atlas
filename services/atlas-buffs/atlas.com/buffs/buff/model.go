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
	noExpiry  bool

	correlationId string
}

func (m Model) SourceId() int32 {
	return m.sourceId
}

func (m Model) Level() byte {
	return m.level
}

// NoExpiry reports whether this buff never expires on its own (e.g. the
// HOMING_BEACON lock). The expiration ticker must never reap it; it is
// removed only by explicit cancel flows.
func (m Model) NoExpiry() bool {
	return m.noExpiry
}

// CorrelationId names the thing that granted this buff, when something other
// than a skill did — an event occurrence id, today. It is opaque to
// atlas-buffs: the service stores it, echoes it, and matches it for equality so
// the granter can cancel exactly what it granted (FR-A12), without knowing what
// an event is.
func (m Model) CorrelationId() string { return m.correlationId }

func (m Model) Expired() bool {
	if m.noExpiry {
		return false
	}
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

// WithStatAmount returns a copy of the buff with the amount of the stat of
// the given type replaced, preserving identity (id, sourceId, level,
// duration), the other stats, the noExpiry flag, and the ORIGINAL
// createdAt/expiresAt — value updates must not extend the buff's lifetime.
// The second return is false when the buff has no stat of that type.
func (m Model) WithStatAmount(statType string, amount int32) (Model, bool) {
	found := false
	changes := make([]stat.Model, 0, len(m.changes))
	for _, c := range m.changes {
		if c.Type() == statType {
			changes = append(changes, stat.NewStat(statType, amount))
			found = true
		} else {
			changes = append(changes, c)
		}
	}
	if !found {
		return Model{}, false
	}
	return Model{
		id:            m.id,
		sourceId:      m.sourceId,
		level:         m.level,
		duration:      m.duration,
		changes:       changes,
		createdAt:     m.createdAt,
		expiresAt:     m.expiresAt,
		noExpiry:      m.noExpiry,
		correlationId: m.correlationId,
	}, true
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id            uuid.UUID    `json:"id"`
		SourceId      int32        `json:"sourceId"`
		Level         byte         `json:"level"`
		Duration      int32        `json:"duration"`
		Changes       []stat.Model `json:"changes"`
		CreatedAt     time.Time    `json:"createdAt"`
		ExpiresAt     time.Time    `json:"expiresAt"`
		NoExpiry      bool         `json:"noExpiry,omitempty"`
		CorrelationId string       `json:"correlationId,omitempty"`
	}{
		Id:            m.id,
		SourceId:      m.sourceId,
		Level:         m.level,
		Duration:      m.duration,
		Changes:       m.changes,
		CreatedAt:     m.createdAt,
		ExpiresAt:     m.expiresAt,
		NoExpiry:      m.noExpiry,
		CorrelationId: m.correlationId,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id            uuid.UUID    `json:"id"`
		SourceId      int32        `json:"sourceId"`
		Level         byte         `json:"level"`
		Duration      int32        `json:"duration"`
		Changes       []stat.Model `json:"changes"`
		CreatedAt     time.Time    `json:"createdAt"`
		ExpiresAt     time.Time    `json:"expiresAt"`
		NoExpiry      bool         `json:"noExpiry,omitempty"`
		CorrelationId string       `json:"correlationId,omitempty"`
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
	m.noExpiry = aux.NoExpiry
	m.correlationId = aux.CorrelationId
	return nil
}

var (
	ErrInvalidDuration = errors.New("duration must be positive")
	ErrEmptyChanges    = errors.New("changes cannot be empty")
)

func NewBuff(sourceId int32, level byte, duration int32, changes []stat.Model, correlationId string) (Model, error) {
	if duration <= 0 {
		return Model{}, ErrInvalidDuration
	}
	if len(changes) == 0 {
		return Model{}, ErrEmptyChanges
	}
	return Model{
		id:            uuid.New(),
		sourceId:      sourceId,
		level:         level,
		duration:      duration,
		changes:       changes,
		createdAt:     time.Now(),
		expiresAt:     time.Now().Add(time.Duration(duration) * time.Millisecond),
		correlationId: correlationId,
	}, nil
}

// NewNoExpiryBuff builds a buff that never expires on its own. duration is 0
// and expiresAt is the zero time; Expired() short-circuits on the flag so the
// zero expiresAt is never consulted (FR-2.4).
func NewNoExpiryBuff(sourceId int32, level byte, changes []stat.Model, correlationId string) (Model, error) {
	if len(changes) == 0 {
		return Model{}, ErrEmptyChanges
	}
	return Model{
		id:            uuid.New(),
		sourceId:      sourceId,
		level:         level,
		duration:      0,
		changes:       changes,
		createdAt:     time.Now(),
		expiresAt:     time.Time{},
		noExpiry:      true,
		correlationId: correlationId,
	}, nil
}
