package account

import (
	"errors"
	"github.com/google/uuid"
	"sync"
	"time"
)

var instance *Registry
var once sync.Once

func Get() *Registry {
	once.Do(func() {
		instance = &Registry{
			lock:         sync.RWMutex{},
			sessions:     make(map[AccountKey]map[ServiceKey]StateValue),
			sessionLocks: map[AccountKey]sync.RWMutex{},
		}
	})
	return instance
}

type AccountKey struct {
	TenantId  uuid.UUID
	AccountId uint32
}

type Service string

const (
	ServiceLogin   = "LOGIN"
	ServiceChannel = "CHANNEL"
)

type StateValue struct {
	State     State
	UpdatedAt time.Time
}

type ServiceKey struct {
	SessionId uuid.UUID
	Service   Service
}

type Registry struct {
	lock         sync.RWMutex
	sessions     map[AccountKey]map[ServiceKey]StateValue
	sessionLocks map[AccountKey]sync.RWMutex
}

func (l *Registry) MaximalState(key AccountKey) State {
	var sl sync.RWMutex
	var ok bool
	if sl, ok = l.sessionLocks[key]; !ok {
		l.lock.Lock()
		l.sessionLocks[key] = sync.RWMutex{}
		l.sessions[key] = make(map[ServiceKey]StateValue)
		l.lock.Unlock()
		return 0
	}
	sl.RLock()
	defer sl.RUnlock()
	var maximalState = uint8(99)
	if len(l.sessions[key]) == 0 {
		return 0
	}

	if states, ok := l.sessions[key]; ok {
		for _, state := range states {
			if uint8(state.State) < maximalState {
				maximalState = uint8(state.State)
			}
		}
	}
	return State(maximalState)
}

func (l *Registry) IsLoggedIn(key AccountKey) bool {
	return l.MaximalState(key) > 0
}

func (l *Registry) Login(key AccountKey, sk ServiceKey) error {
	var sl sync.RWMutex
	var ok bool
	if sl, ok = l.sessionLocks[key]; !ok {
		l.lock.Lock()
		l.sessionLocks[key] = sync.RWMutex{}
		l.sessions[key] = make(map[ServiceKey]StateValue)
		l.lock.Unlock()
		//sl = l.sessionLocks[key]
	}
	sl.Lock()
	defer sl.Unlock()
	if states, ok := l.sessions[key]; ok {
		if sk.Service == ServiceLogin {
			for _, state := range states {
				if state.State > 0 {
					return errors.New("already logged in")
				}
			}
			states[sk] = StateValue{State: StateLoggedIn, UpdatedAt: time.Now()}
			return nil
		} else if sk.Service == ServiceChannel {
			for _, state := range states {
				if state.State > 1 {
					l.sessions[key][sk] = StateValue{State: StateLoggedIn, UpdatedAt: time.Now()}
					return nil
				}
			}
			return errors.New("no other service transitioning")
		}
		return errors.New("undefined service")
	}
	return errors.New("unexpected error")
}

func (l *Registry) Transition(key AccountKey, sk ServiceKey) error {
	var sl sync.RWMutex
	var ok bool
	if sl, ok = l.sessionLocks[key]; !ok {
		l.lock.Lock()
		l.sessionLocks[key] = sync.RWMutex{}
		l.sessions[key] = make(map[ServiceKey]StateValue)
		l.lock.Unlock()
		//sl = l.sessionLocks[key]
	}
	sl.Lock()
	defer sl.Unlock()
	if states, ok := l.sessions[key]; ok {
		if state, ok := states[sk]; ok {
			if state.State > 0 {
				l.sessions[key][sk] = StateValue{State: StateTransition, UpdatedAt: time.Now()}
				return nil
			}
		}
	}
	return errors.New("not logged in")
}

func (l *Registry) ExpireTransition(key AccountKey) {
	var sl sync.RWMutex
	var ok bool
	if sl, ok = l.sessionLocks[key]; !ok {
		return
	}
	sl.Lock()
	defer sl.Unlock()
	if states, ok := l.sessions[key]; ok {
		for sk, state := range states {
			if state.State == 2 {
				delete(states, sk)
			}
		}
	}
}

func (l *Registry) Logout(key AccountKey, sk ServiceKey) bool {
	var sl sync.RWMutex
	var ok bool
	if sl, ok = l.sessionLocks[key]; !ok {
		return false
	}
	sl.Lock()
	defer sl.Unlock()
	if states, ok := l.sessions[key]; ok {
		if states[sk].State != 2 {
			delete(states, sk)
			return true
		}
	}
	return false
}

func (l *Registry) GetExpiredInTransition(timeout time.Duration) []AccountKey {
	l.lock.Lock()
	defer l.lock.Unlock()
	accounts := make([]AccountKey, 0)
	for account, session := range l.sessions {
		for _, state := range session {
			if state.State == StateTransition && time.Now().Sub(state.UpdatedAt) > timeout {
				accounts = append(accounts, account)
			}
		}
	}
	return accounts
}
