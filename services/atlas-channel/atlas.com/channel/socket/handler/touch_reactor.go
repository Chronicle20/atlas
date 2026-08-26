package handler

import (
	"atlas-channel/reactor"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	reactor2 "github.com/Chronicle20/atlas/libs/atlas-packet/reactor/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func TouchReactorHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := reactor2.TouchingRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		err := reactor.NewProcessor(l, ctx).Touch(s.Field(), p.Oid(), s.CharacterId(), p.Touching())
		if err != nil {
			l.WithError(err).Errorf("Unable to send touch command for reactor [%d].", p.Oid())
		}
	}
}
