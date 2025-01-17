package session

import (
	as "atlas-login/account/session"
	"atlas-login/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"net"
)

func Create(l logrus.FieldLogger, r *Registry) func(t tenant.Model, locale byte) func(sessionId uuid.UUID, conn net.Conn) {
	return func(t tenant.Model, locale byte) func(sessionId uuid.UUID, conn net.Conn) {
		return func(sessionId uuid.UUID, conn net.Conn) {
			fl := l.WithField("session", sessionId)
			fl.Debugf("Creating session.")
			s := NewSession(sessionId, t, locale, conn)
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
			s, ok := r.Get(tenant.Id(), sessionId)
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

func DestroyAll(l logrus.FieldLogger, ctx context.Context, r *Registry) model.Operator[tenant.Model] {
	return func(t tenant.Model) error {
		tctx := tenant.WithContext(ctx, t)
		return model.ForEachSlice(AllInTenantProvider(t), Destroy(l, tctx, r))
	}
}

func DestroyByIdWithSpan(l logrus.FieldLogger) func(ctx context.Context) func(r *Registry) func(sessionId uuid.UUID) {
	return func(ctx context.Context) func(r *Registry) func(sessionId uuid.UUID) {
		return func(r *Registry) func(sessionId uuid.UUID) {
			return func(sessionId uuid.UUID) {
				sctx, span := otel.GetTracerProvider().Tracer("atlas-login").Start(ctx, "session-destroy")
				defer span.End()
				DestroyById(l, sctx, r)(sessionId)
			}
		}
	}
}

func DestroyById(l logrus.FieldLogger, ctx context.Context, r *Registry) func(sessionId uuid.UUID) {
	t := tenant.MustFromContext(ctx)
	return func(sessionId uuid.UUID) {
		s, ok := r.Get(t.Id(), sessionId)
		if !ok {
			return
		}
		Destroy(l, ctx, r)(s)
	}
}

func Destroy(l logrus.FieldLogger, ctx context.Context, r *Registry) model.Operator[Model] {
	t := tenant.MustFromContext(ctx)
	pi := producer.ProviderImpl(l)(ctx)
	return func(s Model) error {
		l.WithField("session", s.SessionId().String()).Debugf("Destroying session.")
		r.Remove(t.Id(), s.SessionId())
		s.Disconnect()
		as.Destroy(l, pi)(s.SessionId(), s.AccountId())
		return pi(EnvEventTopicSessionStatus)(destroyedStatusEventProvider(s.SessionId(), s.AccountId()))
	}
}
