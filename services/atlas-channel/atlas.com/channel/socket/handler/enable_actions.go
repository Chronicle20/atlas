package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"
)

// enableActions is a package-local alias for session.EnableActions, kept so the
// many call sites in this package read unchanged. The implementation — and the
// contract for when it must and must not be sent — lives in
// session/enable_actions.go, because the Kafka consumers need it too.
func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return session.EnableActions(l)
}
