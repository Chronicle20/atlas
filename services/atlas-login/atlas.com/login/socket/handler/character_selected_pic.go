package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/channel"
	"atlas-login/session"
	"atlas-login/socket/model"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterSelectedPicHandle = "CharacterSelectedPicHandle"

func CharacterSelectedPicHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	t := tenant.MustFromContext(ctx)
	serverIpFunc := session.Announce(l)(wp)(writer.ServerIP)
	return func(s session.Model, r *request.Reader) {
		pic := r.ReadAsciiString()
		characterId := r.ReadUint32()

		if t.Region() == "GMS" {
			_ = r.ReadAsciiString() // sMacAddressWithHDDSerial
			_ = r.ReadAsciiString() // sMacAddressWithHDDSerial2
		}
		l.Debugf("Character [%d] selected for login to channel [%d:%d].", characterId, s.WorldId(), s.ChannelId())

		ap := account.NewProcessor(l, ctx)
		a, err := ap.GetById(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve account [%d] for PIC validation.", s.AccountId())
			err = serverIpFunc(s, writer.ServerIPBodySimpleError(l)(writer.ServerIPCodeServerUnderInspection))
			if err != nil {
				l.WithError(err).Errorf("Unable to write server ip response due to error.")
			}
			return
		}

		if a.PIC() != pic {
			l.Debugf("Incorrect PIC for account [%d].", s.AccountId())
			_, limitReached, _ := ap.RecordPicAttempt(s.AccountId(), false)
			if limitReached {
				l.Warnf("Account [%d] has exceeded PIC attempt limit. Terminating session.", s.AccountId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}
			err = serverIpFunc(s, writer.ServerIPBodySimpleError(l)(writer.ServerIPCodeIncorrectPassword))
			if err != nil {
				l.WithError(err).Errorf("Unable to write server ip response due to error.")
			}
			return
		}

		ap.RecordPicAttempt(s.AccountId(), true)

		c, err := channel.NewProcessor(l, ctx).GetById(s.Channel())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve channel information being logged in to.")
			err = serverIpFunc(s, writer.ServerIPBodySimpleError(l)(writer.ServerIPCodeServerUnderInspection))
			if err != nil {
				l.WithError(err).Errorf("Unable to write server ip response due to error.")
				return
			}
			return
		}

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelSelect{IPAddress: c.IpAddress(), Port: uint16(c.Port()), CharacterId: characterId})
		if err != nil {
			return
		}
	}
}
