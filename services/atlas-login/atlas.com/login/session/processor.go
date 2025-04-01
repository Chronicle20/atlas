package session

import (
	"atlas-login/kafka/producer"
	"atlas-login/socket/writer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

func AllInTenantProvider(tenant tenant.Model) model.Provider[[]Model] {
	return func() ([]Model, error) {
		return GetRegistry().GetInTenant(tenant.Id()), nil
	}
}

func ByIdModelProvider(tenant tenant.Model) func(sessionId uuid.UUID) model.Provider[Model] {
	return func(sessionId uuid.UUID) model.Provider[Model] {
		return func() (Model, error) {
			s, ok := GetRegistry().Get(tenant.Id(), sessionId)
			if !ok {
				return Model{}, errors.New("not found")
			}
			return s, nil
		}
	}
}

func IfPresentById(tenant tenant.Model) func(sessionId uuid.UUID, f model.Operator[Model]) {
	return func(sessionId uuid.UUID, f model.Operator[Model]) {
		s, err := ByIdModelProvider(tenant)(sessionId)()
		if err != nil {
			return
		}
		_ = f(s)
	}
}

func Announce(l logrus.FieldLogger) func(writerProducer writer.Producer) func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
	return func(writerProducer writer.Producer) func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
		return func(writerName string) func(s Model, bodyProducer writer.BodyProducer) error {
			return func(s Model, bodyProducer writer.BodyProducer) error {
				w, err := writerProducer(l, writerName)
				if err != nil {
					return err
				}
				return s.announceEncrypted(w(l)(bodyProducer))
			}
		}
	}
}

func SetAccountId(accountId uint32) func(tenantId uuid.UUID, id uuid.UUID) Model {
	return func(tenantId uuid.UUID, id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(tenantId, id); ok {
			s = s.setAccountId(accountId)
			GetRegistry().Update(tenantId, s)
			return s
		}
		return s
	}
}

func UpdateLastRequest() func(tenantId uuid.UUID, id uuid.UUID) Model {
	return func(tenantId uuid.UUID, id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(tenantId, id); ok {
			s = s.updateLastRequest()
			GetRegistry().Update(tenantId, s)
			return s
		}
		return s
	}
}

func SetWorldId(worldId byte) func(tenantId uuid.UUID, id uuid.UUID) Model {
	return func(tenantId uuid.UUID, id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(tenantId, id); ok {
			s = s.setWorldId(worldId)
			GetRegistry().Update(tenantId, s)
			return s
		}
		return s
	}
}

func SetChannelId(channelId byte) func(tenantId uuid.UUID, id uuid.UUID) Model {
	return func(tenantId uuid.UUID, id uuid.UUID) Model {
		s := Model{}
		var ok bool
		if s, ok = GetRegistry().Get(tenantId, id); ok {
			s = s.setChannelId(channelId)
			GetRegistry().Update(tenantId, s)
			return s
		}
		return s
	}
}

func SessionCreated(kp producer.Provider, tenant tenant.Model) func(s Model) {
	return func(s Model) {
		_ = kp(EnvEventTopicSessionStatus)(createdStatusEventProvider(s.SessionId(), s.AccountId()))
	}
}

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		ctx, span := otel.GetTracerProvider().Tracer("atlas-login").Start(context.Background(), "teardown")
		defer span.End()

		tenant.ForAll(DestroyAll(l, ctx, GetRegistry()))
	}
}
