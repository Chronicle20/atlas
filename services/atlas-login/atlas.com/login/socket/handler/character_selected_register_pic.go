package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/channel"
	"atlas-login/session"
	"atlas-login/socket/model"
	"atlas-login/socket/writer"
	"context"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func RegisterPicHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := loginpkt.CharacterSelectRegisterPic{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		a, err := account.NewProcessor(l, ctx).GetById(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Failed to get account by id [%d].", s.AccountId())
			//TODO
			return
		}
		if a.PIC() != "" {
			l.Warnf("Account [%d] already has PIC.", s.AccountId())
			//TODO
			return
		}
		err = account.NewProcessor(l, ctx).UpdatePic(s.AccountId(), p.Pic())
		if err != nil {
			l.WithError(err).Errorf("Unable to register PIC for account [%d].", s.AccountId())
		}

		c, err := channel.NewProcessor(l, ctx).GetById(s.Channel())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve channel information being logged in to.")
			err = session.Announce(l)(ctx)(wp)(loginpkt.ServerIPWriter)(writer.ServerIPBodySimpleError(writer.ServerIPCodeServerUnderInspection))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write server ip response due to error.")
				return
			}
			return
		}

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelSelect{IPAddress: c.IpAddress(), Port: uint16(c.Port()), CharacterId: p.CharacterId()})
		if err != nil {
			return
		}
	}
}
