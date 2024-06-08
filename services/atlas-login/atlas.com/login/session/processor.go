package session

import (
	"errors"
	"github.com/google/uuid"
)

func Announce(b []byte) func(s Model) error {
	return func(s Model) error {
		if l, ok := GetRegistry().GetLock(s.SessionId()); ok {
			l.Lock()
			err := s.announceEncrypted(b)
			l.Unlock()
			return err
		}
		return errors.New("invalid session")
	}
}

func UpdateLastRequest() func(id uuid.UUID) Model {
	return func(id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(id); ok {
			s = s.updateLastRequest()
			GetRegistry().Update(s)
			return s
		}
		return s
	}
}

func SetWorldId(worldId byte) func(id uuid.UUID) Model {
	return func(id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(id); ok {
			s = s.setWorldId(worldId)
			GetRegistry().Update(s)
			return s
		}
		return s
	}
}

func SetChannelId(channelId byte) func(id uuid.UUID) Model {
	return func(id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(id); ok {
			s = s.setChannelId(channelId)
			GetRegistry().Update(s)
			return s
		}
		return s
	}
}
