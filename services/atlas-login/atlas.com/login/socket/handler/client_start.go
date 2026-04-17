package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ClientStartHandle = "ClientStartHandle"

func ClientStartHandleFunc(l logrus.FieldLogger, _ context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		l.Debugf("Client has started.")
	}
}
