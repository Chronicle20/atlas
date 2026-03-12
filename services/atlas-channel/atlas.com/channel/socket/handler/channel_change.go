package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/channel"
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	channel3 "github.com/Chronicle20/atlas-packet/channel"

	channel2 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func ChannelChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := channel3.ChannelChangeRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		ch, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d].", s.CharacterId())
			return
		}
		if ch.Hp() == 0 {
			l.Warnf("Character [%d] attempting to change channel when dead.", s.CharacterId())
			return
		}

		// TODO verify not in mini dungeon

		c, err := channel.NewProcessor(l, ctx).GetById(channel2.NewModel(s.WorldId(), p.ChannelId()))
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve channel information being logged in to.")
			// TODO send server notice.
			return
		}

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelChange{IPAddress: c.IpAddress(), Port: uint16(c.Port())})
		if err != nil {
			_ = session.NewProcessor(l, ctx).Destroy(s)
		}
	}
}
