package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/character"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	character2 "github.com/Chronicle20/atlas-packet/socket"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterLoggedInHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.ChannelConnect{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		c, err := character.NewProcessor(l, ctx).GetById()(p.CharacterId())
		if err != nil {
			return
		}

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), c.AccountId(), 1, model2.SetField{CharacterId: p.CharacterId()})
		if err != nil {
			_ = session.NewProcessor(l, ctx).Destroy(s)
		}
		return
	}
}
