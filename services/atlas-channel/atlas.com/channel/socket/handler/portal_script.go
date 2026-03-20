package handler

import (
	"atlas-channel/portal"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	portal2 "github.com/Chronicle20/atlas-packet/portal/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func PortalScriptHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := portal2.Script{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		_ = portal.NewProcessor(l, ctx).Enter(s.Field(), p.PortalName(), s.CharacterId())
	}
}
