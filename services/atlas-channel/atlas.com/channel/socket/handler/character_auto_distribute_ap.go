package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	character2 "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterAutoDistributeApHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.AutoDistributeAp{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		var distributes = make([]character.DistributePacket, 0)
		for _, d := range p.Distributes() {
			distributes = append(distributes, character.DistributePacket{Flag: d.Flag, Value: d.Value})
		}
		_ = character.NewProcessor(l, ctx).RequestDistributeAp(s.Field(), s.CharacterId(), p.UpdateTime(), distributes)
	}
}
