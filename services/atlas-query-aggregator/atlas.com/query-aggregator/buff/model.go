package buff

import "time"

// Model represents an active buff on a character
type Model struct {
	sourceId  int32
	duration  int32
	createdAt time.Time
	expiresAt time.Time
}

// NewModel creates a new buff model
func NewModel(sourceId int32, duration int32, createdAt time.Time, expiresAt time.Time) Model {
	return Model{
		sourceId:  sourceId,
		duration:  duration,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}
}

// SourceId returns the buff source ID (skill/item that applied the buff)
func (m Model) SourceId() int32 {
	return m.sourceId
}

// Duration returns the buff duration in seconds
func (m Model) Duration() int32 {
	return m.duration
}

// CreatedAt returns when the buff was applied
func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

// ExpiresAt returns when the buff expires
func (m Model) ExpiresAt() time.Time {
	return m.expiresAt
}

// IsActive returns true if the buff has not expired
func (m Model) IsActive() bool {
	return time.Now().Before(m.expiresAt)
}
