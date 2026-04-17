package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	loginCB "github.com/Chronicle20/atlas/libs/atlas-packet/login/clientbound"
	loginSB "github.com/Chronicle20/atlas/libs/atlas-packet/login/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func WorldSelectHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := loginSB.ServerSelect{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		err := session.Announce(l)(ctx)(wp)(loginCB.ServerLoadWriter)(writer.ServerLoadBody(writer.ServerLoadCodeOk))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to issue request server load")
		}
	}
}
