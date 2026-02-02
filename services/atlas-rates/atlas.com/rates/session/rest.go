package session

import (
	"time"
)

// SessionRestModel represents a gameplay session from atlas-character
type SessionRestModel struct {
	Id          string     `json:"-"`
	CharacterId uint32     `json:"characterId"`
	WorldId     byte       `json:"worldId"`
	ChannelId   byte       `json:"channelId"`
	LoginTime   time.Time  `json:"loginTime"`
	LogoutTime  *time.Time `json:"logoutTime,omitempty"`
}

func (r SessionRestModel) GetName() string {
	return "sessions"
}

func (r SessionRestModel) GetID() string {
	return r.Id
}

func (r *SessionRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// IsActive returns true if the session is still active (no logout time)
func (r SessionRestModel) IsActive() bool {
	return r.LogoutTime == nil
}

// Duration returns the session duration
func (r SessionRestModel) Duration(now time.Time) time.Duration {
	endTime := now
	if r.LogoutTime != nil {
		endTime = *r.LogoutTime
	}
	return endTime.Sub(r.LoginTime)
}

// OverlapsWith calculates the overlap duration between this session and [start, end]
func (r SessionRestModel) OverlapsWith(start, end time.Time) time.Duration {
	sessionEnd := end
	if r.LogoutTime != nil && r.LogoutTime.Before(end) {
		sessionEnd = *r.LogoutTime
	}

	overlapStart := r.LoginTime
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

// PlaytimeRestModel represents a playtime computation response
type PlaytimeRestModel struct {
	Id            string `json:"-"`
	CharacterId   uint32 `json:"characterId"`
	TotalSeconds  int64  `json:"totalSeconds"`
	FormattedTime string `json:"formattedTime"`
}

func (r PlaytimeRestModel) GetName() string {
	return "playtime"
}

func (r PlaytimeRestModel) GetID() string {
	return r.Id
}

func (r *PlaytimeRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
