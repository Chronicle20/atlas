package account

import (
	"errors"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

var instance *Registry
var once sync.Once

func Get() *Registry {
	once.Do(func() {
		instance = &Registry{
			lock:     sync.RWMutex{},
			sessions: make(map[AccountKey]map[ServiceKey]StateValue),
		}
	})
	return instance
}

type AccountKey struct {
	Tenant    tenant.Model
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
	lock     sync.RWMutex
	sessions map[AccountKey]map[ServiceKey]StateValue
}

func (l *Registry) GetStates(key AccountKey) map[ServiceKey]StateValue {
	l.lock.RLock()
	defer l.lock.RUnlock()

	var states map[ServiceKey]StateValue
	var ok bool
	if states, ok = l.sessions[key]; !ok {
		return map[ServiceKey]StateValue{}
	}

	return states
}

func (l *Registry) MaximalState(key AccountKey) State {
	l.lock.RLock()
	defer l.lock.RUnlock()

	var states map[ServiceKey]StateValue
	var ok bool
	if states, ok = l.sessions[key]; !ok {
		return StateNotLoggedIn
	}

	var maximalState = uint8(99)
	if len(states) == 0 {
		return StateNotLoggedIn
	}

	for _, state := range states {
		if uint8(state.State) < maximalState {
			maximalState = uint8(state.State)
		}
	}
	return State(maximalState)
}

func (l *Registry) IsLoggedIn(key AccountKey) bool {
	return l.MaximalState(key) > 0
}

func (l *Registry) Login(key AccountKey, sk ServiceKey) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	var states map[ServiceKey]StateValue
	var ok bool
	if states, ok = l.sessions[key]; !ok {
		l.sessions[key] = make(map[ServiceKey]StateValue)
		states = l.sessions[key]
	}

	if sk.Service == ServiceLogin {
		for _, state := range states {
			if state.State > 0 {
				return errors.New("already logged in")
			}
		}
		states[sk] = StateValue{State: StateLoggedIn, UpdatedAt: time.Now()}
		return nil
	} else if sk.Service == ServiceChannel {
		// Remove ServiceLogin states if we've successfully moved to a ServiceChannel.
		var transition = false
		for tk, ts := range states {
			if ts.State > 1 {
				transition = true
			}
			delete(states, tk)
		}

		if transition {
			l.sessions[key][sk] = StateValue{State: StateLoggedIn, UpdatedAt: time.Now()}
			return nil
		}
		return errors.New("no other service transitioning")
	}
	return errors.New("undefined service")
}

func (l *Registry) Transition(key AccountKey, sk ServiceKey) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	var states map[ServiceKey]StateValue
	var ok bool
	if states, ok = l.sessions[key]; !ok {
		l.sessions[key] = make(map[ServiceKey]StateValue)
		states = l.sessions[key]
	}

	if state, ok := states[sk]; ok {
		if state.State > 0 {
			l.sessions[key][sk] = StateValue{State: StateTransition, UpdatedAt: time.Now()}
			return nil
		}
	}
	return errors.New("not logged in")
}

func (l *Registry) ExpireTransition(key AccountKey, timeout time.Duration) {
	l.lock.Lock()
	defer l.lock.Unlock()

	var states map[ServiceKey]StateValue
	var ok bool
	if states, ok = l.sessions[key]; !ok {
		l.sessions[key] = make(map[ServiceKey]StateValue)
		states = l.sessions[key]
	}

	for sk, state := range states {
		if state.State == StateTransition && time.Now().Sub(state.UpdatedAt) > timeout {
			delete(states, sk)
		}
	}
}

func (l *Registry) Logout(key AccountKey, sk ServiceKey) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	var states map[ServiceKey]StateValue
	var ok bool
	if states, ok = l.sessions[key]; !ok {
		l.sessions[key] = make(map[ServiceKey]StateValue)
		states = l.sessions[key]
	}

	if states[sk].State != 2 {
		delete(states, sk)
		return true
	}
	return false
}

func (l *Registry) Terminate(key AccountKey) bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.sessions[key] = make(map[ServiceKey]StateValue)
	return true
}

func (l *Registry) GetExpiredInTransition(timeout time.Duration) []AccountKey {
	l.lock.RLock()
	defer l.lock.RUnlock()

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

func (l *Registry) Tenants() []tenant.Model {
	l.lock.Lock()
	defer l.lock.Unlock()
	var tenants = make([]tenant.Model, 0)
	for ak := range l.sessions {
		tenants = append(tenants, ak.Tenant)
	}
	return tenants
}
