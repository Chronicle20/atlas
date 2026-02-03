package session

import (
	"sync"
)

// Session represents an online session
type Session struct {
	CharacterId uint32
	AccountId   uint32
	WorldId     byte
	ChannelId   byte
}

// Tracker tracks online sessions for periodic expiration checks
type Tracker struct {
	mu       sync.RWMutex
	sessions map[uint32]Session // keyed by characterId
}

var tracker *Tracker
var once sync.Once

// GetTracker returns the singleton session tracker
func GetTracker() *Tracker {
	once.Do(func() {
		tracker = &Tracker{
			sessions: make(map[uint32]Session),
		}
	})
	return tracker
}

// Add adds or updates a session
func (t *Tracker) Add(characterId, accountId uint32, worldId, channelId byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sessions[characterId] = Session{
		CharacterId: characterId,
		AccountId:   accountId,
		WorldId:     worldId,
		ChannelId:   channelId,
	}
}

// Remove removes a session by character ID
func (t *Tracker) Remove(characterId uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.sessions, characterId)
}

// GetAll returns a snapshot of all sessions
func (t *Tracker) GetAll() []Session {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]Session, 0, len(t.sessions))
	for _, s := range t.sessions {
		result = append(result, s)
	}
	return result
}

// Get returns a session by character ID
func (t *Tracker) Get(characterId uint32) (Session, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	s, ok := t.sessions[characterId]
	return s, ok
}

// Count returns the number of tracked sessions
func (t *Tracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.sessions)
}
