package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const PongHandle = "PongHandle"

func PongHandleFunc(l logrus.FieldLogger, _ context.Context, _ writer.Producer) func(s session.Model, r *request.Reader) {
	return func(s session.Model, r *request.Reader) {
		l.Debugf("Received PONG from account [%d].", s.AccountId())
	}
}
