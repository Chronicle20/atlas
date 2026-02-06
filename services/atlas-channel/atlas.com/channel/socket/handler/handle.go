package handler

import (
	"atlas-channel/account"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

type MessageValidator func(l logrus.FieldLogger, ctx context.Context) func(s session.Model) bool

const NoOpValidator = "NoOpValidator"

func NoOpValidatorFunc(_ logrus.FieldLogger, _ context.Context) func(_ session.Model) bool {
	return func(_ session.Model) bool {
		return true
	}
}

const LoggedInValidator = "LoggedInValidator"

func LoggedInValidatorFunc(l logrus.FieldLogger, ctx context.Context) func(s session.Model) bool {
	return func(s session.Model) bool {
		v := account.NewProcessor(l, ctx).IsLoggedIn(s.AccountId())
		if !v {
			l.Errorf("Attempting to process a request when the account [%d] is not logged in. Terminating session.", s.AccountId())
			session.NewProcessor(l, ctx).Destroy(s)
		}
		return v
	}
}

type MessageHandler func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})

const NoOpHandler = "NoOpHandler"

func NoOpHandlerFunc(_ logrus.FieldLogger, _ context.Context, _ writer.Producer) func(_ session.Model, _ *request.Reader, _ map[string]interface{}) {
	return func(_ session.Model, _ *request.Reader, _ map[string]interface{}) {
	}
}

type Adapter func(name string, v MessageValidator, h MessageHandler, readerOptions map[string]interface{}) request.Handler

func AdaptHandler(l logrus.FieldLogger) func(t tenant.Model, wp writer.Producer) Adapter {
	return func(t tenant.Model, wp writer.Producer) Adapter {
		return func(name string, v MessageValidator, h MessageHandler, readerOptions map[string]interface{}) request.Handler {
			return func(sessionId uuid.UUID, r request.Reader) {
				fl := l.WithField("session", sessionId.String())

				sctx, span := otel.GetTracerProvider().Tracer("atlas-channel").Start(context.Background(), name)
				sl := fl.WithField("trace.id", span.SpanContext().TraceID().String()).WithField("span.id", span.SpanContext().SpanID().String())
				defer span.End()

				tctx := tenant.WithContext(sctx, t)

				sp := session.NewProcessor(l, tctx)
				sp.IfPresentById(sessionId, func(s session.Model) error {
					if v(sl, tctx)(s) {
						h(sl, tctx, wp)(s, &r, readerOptions)
						_ = sp.UpdateLastRequest(s.SessionId())
					}
					return nil
				})
			}
		}
	}
}
