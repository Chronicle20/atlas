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

func CharacterChairPortableHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.ChairPortable{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = chair.NewProcessor(l, ctx).Use(s.Field(), chair2.TypePortable, p.ItemId(), s.CharacterId())
	}
}
