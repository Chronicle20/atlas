package handler

import (
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func ServerStatusHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := loginpkt.ServerStatusRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		cs := world.NewProcessor(l, ctx).GetCapacityStatus(p.WorldId())
		err := session.Announce(l)(ctx)(wp)(loginpkt.ServerStatusWriter)(loginpkt.NewServerStatus(uint16(cs)).Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to issue world capacity status information")
		}
	}
}
