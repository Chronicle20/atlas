package session

import (
	"atlas-login/tenant"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
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

func Decrypt(_ logrus.FieldLogger, r *Registry) func(hasMapleEncryption bool) func(sessionId uuid.UUID, input []byte) []byte {
	return func(hasMapleEncryption bool) func(sessionId uuid.UUID, input []byte) []byte {
		return func(sessionId uuid.UUID, input []byte) []byte {
			s, ok := r.Get(sessionId)
			if !ok {
				return input
			}
			if s.ReceiveAESOFB() == nil {
				return input
			}
			return s.ReceiveAESOFB().Decrypt(true, hasMapleEncryption)(input)
		}
	}
}

func DestroyAll(l logrus.FieldLogger, span opentracing.Span, r *Registry) {
	for _, s := range r.GetAll() {
		Destroy(l, span, r)(s)
	}
}

func DestroyByIdWithSpan(l logrus.FieldLogger, r *Registry) func(sessionId uuid.UUID) {
	return func(sessionId uuid.UUID) {
		span := opentracing.StartSpan("session_destroy")
		defer span.Finish()
		DestroyById(l, span, r)(sessionId)
	}
}

func DestroyById(l logrus.FieldLogger, span opentracing.Span, r *Registry) func(sessionId uuid.UUID) {
	return func(sessionId uuid.UUID) {
		s, ok := r.Get(sessionId)
		if !ok {
			return
		}
		Destroy(l, span, r)(s)
	}
}

func Destroy(l logrus.FieldLogger, span opentracing.Span, r *Registry) func(Model) {
	return func(s Model) {
		l.Debugf("Destroying session %d.", s.SessionId())
		r.Remove(s.SessionId())
	}
}
