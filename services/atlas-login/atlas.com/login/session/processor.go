package session

import (
	"atlas-login/socket/writer"
	"atlas-login/tenant"
	"errors"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func Announce(writerProducer writer.Producer) func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
	return func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
		return func(s Model, bodyProducer writer.BodyProducer) error {
			w, err := writerProducer(writerName)
			if err != nil {
				return err
			}

			if lock, ok := GetRegistry().GetLock(s.SessionId()); ok {
				lock.Lock()
				err = s.announceEncrypted(w(bodyProducer))
				lock.Unlock()
				return err
			}
			return errors.New("invalid session")
		}
	}
}

func SetAccountId(accountId uint32) func(id uuid.UUID) Model {
	return func(id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(id); ok {
			s = s.setAccountId(accountId)
			GetRegistry().Update(s)
			return s
		}
		return s
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

func SessionCreated(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(sessionId uuid.UUID, accountId uint32) {
	return func(sessionId uuid.UUID, accountId uint32) {
		emitCreatedStatusEvent(l, span, tenant)(sessionId, accountId)
	}
}
