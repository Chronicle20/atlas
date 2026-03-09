package handler

import (
	"atlas-channel/chair"
	chair2 "atlas-channel/kafka/message/chair"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	character2 "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterChairFixedHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	cp := chair.NewProcessor(l, ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.ChairFixed{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		if p.ChairId() == -1 {
			_ = cp.Cancel(s.Field(), s.CharacterId())
			return
		}

		_ = cp.Use(s.Field(), chair2.TypeFixed, uint32(p.ChairId()), s.CharacterId())
	}
}
