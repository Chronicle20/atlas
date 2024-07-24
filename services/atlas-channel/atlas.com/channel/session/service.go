package session

import (
	as "atlas-channel/account/session"
	"atlas-channel/kafka/producer"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"net"
)

func Create(l logrus.FieldLogger, r *Registry, tenant tenant.Model) func(locale byte) func(sessionId uuid.UUID, conn net.Conn) {
	return func(locale byte) func(sessionId uuid.UUID, conn net.Conn) {
		return func(sessionId uuid.UUID, conn net.Conn) {
			fl := l.WithField("session", sessionId)
			fl.Debugf("Creating session.")
			s := NewSession(sessionId, tenant, locale, conn)
			r.Add(s)

			err := s.WriteHello()
			if err != nil {
				fl.WithError(err).Errorf("Unable to write hello packet.")
			}
		}
	}
}

func Decrypt(_ logrus.FieldLogger, r *Registry, tenant tenant.Model) func(hasAes bool, hasMapleEncryption bool) func(sessionId uuid.UUID, input []byte) []byte {
	return func(hasAes bool, hasMapleEncryption bool) func(sessionId uuid.UUID, input []byte) []byte {
		return func(sessionId uuid.UUID, input []byte) []byte {
			s, ok := r.Get(tenant.Id, sessionId)
			if !ok {
				return input
			}
			if s.ReceiveAESOFB() == nil {
				return input
			}
			return s.ReceiveAESOFB().Decrypt(hasAes, hasMapleEncryption)(input)
		}
	}
}

func DestroyAll(l logrus.FieldLogger, span opentracing.Span, r *Registry) model.Operator[uuid.UUID] {
	return func(tenantId uuid.UUID) error {
		for _, s := range r.GetAll() {
			Destroy(l, span, r, tenantId)(s)
		}
		return nil
	}
}

func DestroyByIdWithSpan(l logrus.FieldLogger, r *Registry, tenantId uuid.UUID) func(sessionId uuid.UUID) {
	return func(sessionId uuid.UUID) {
		span := opentracing.StartSpan("session_destroy")
		defer span.Finish()
		DestroyById(l, span, r, tenantId)(sessionId)
	}
}

func DestroyById(l logrus.FieldLogger, span opentracing.Span, r *Registry, tenantId uuid.UUID) func(sessionId uuid.UUID) {
	return func(sessionId uuid.UUID) {
		s, ok := r.Get(tenantId, sessionId)
		if !ok {
			return
		}
		Destroy(l, span, r, tenantId)(s)
	}
}

func Destroy(l logrus.FieldLogger, span opentracing.Span, r *Registry, tenantId uuid.UUID) func(Model) {
	pi := producer.ProviderImpl(l)(span)
	return func(s Model) {
		l.WithField("session", s.SessionId().String()).Debugf("Destroying session.")
		r.Remove(tenantId, s.SessionId())
		s.Disconnect()
		as.Destroy(l, pi)(s.Tenant(), s.AccountId())

		_ = pi(EnvEventTopicSessionStatus)(destroyedStatusEventProvider(s.tenant, s.SessionId(), s.AccountId(), s.CharacterId(), s.WorldId(), s.ChannelId()))
	}
}
