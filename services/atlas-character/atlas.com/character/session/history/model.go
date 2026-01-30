package history

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

type Model struct {
	id          uint64
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	loginTime   time.Time
	logoutTime  *time.Time
}

func (m Model) Id() uint64 {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) LoginTime() time.Time {
	return m.loginTime
}

func (m Model) LogoutTime() *time.Time {
	return m.logoutTime
}

// IsActive returns true if the session is still active (no logout time)
func (m Model) IsActive() bool {
	return m.logoutTime == nil
}

// Duration returns the duration of the session
// For active sessions, it returns duration from login to now
func (m Model) Duration() time.Duration {
	if m.logoutTime != nil {
		return m.logoutTime.Sub(m.loginTime)
	}
	return time.Since(m.loginTime)
}

// OverlapsWith returns the overlap duration between this session and the given time range
// Returns 0 if there's no overlap
func (m Model) OverlapsWith(start, end time.Time) time.Duration {
	sessionEnd := end
	if m.logoutTime != nil {
		sessionEnd = *m.logoutTime
	}

	// Calculate overlap
	overlapStart := m.loginTime
	if start.After(overlapStart) {
		overlapStart = start
	}

	overlapEnd := sessionEnd
	if end.Before(overlapEnd) {
		overlapEnd = end
	}

	if overlapEnd.After(overlapStart) {
		return overlapEnd.Sub(overlapStart)
	}
	return 0
}

func modelFromEntity(e entity) Model {
	return Model{
		id:          e.ID,
		characterId: e.CharacterId,
		worldId:     e.WorldId,
		channelId:   e.ChannelId,
		loginTime:   e.LoginTime,
		logoutTime:  e.LogoutTime,
	}
}
