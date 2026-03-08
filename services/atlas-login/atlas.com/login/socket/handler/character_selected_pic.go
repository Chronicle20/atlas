package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/channel"
	"atlas-login/session"
	"atlas-login/socket/model"
	"atlas-login/socket/writer"
	"context"
	"net"

	"github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterSelectedPicHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := login.CharacterSelectWithPic{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		ap := account.NewProcessor(l, ctx)
		ipAddress := ""
		if addr := s.GetRemoteAddress(); addr != nil {
			if tcpAddr, ok := addr.(*net.TCPAddr); ok {
				ipAddress = tcpAddr.IP.String()
			} else {
				host, _, err := net.SplitHostPort(addr.String())
				if err == nil {
					ipAddress = host
				}
			}
		}

		a, err := ap.GetById(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve account [%d] for PIC validation.", s.AccountId())
			err = session.Announce(l)(ctx)(wp)(writer.ServerIP)(writer.ServerIPBodySimpleError(writer.ServerIPCodeServerUnderInspection))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write server ip response due to error.")
			}
			return
		}

		if a.PIC() != p.Pic() {
			l.Debugf("Incorrect PIC for account [%d].", s.AccountId())
			_, limitReached, _ := ap.RecordPicAttempt(s.AccountId(), false, ipAddress, "")
			if limitReached {
				l.Warnf("Account [%d] has exceeded PIC attempt limit. Terminating session.", s.AccountId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}
			err = session.Announce(l)(ctx)(wp)(writer.ServerIP)(writer.ServerIPBodySimpleError(writer.ServerIPCodeIncorrectPassword))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write server ip response due to error.")
			}
			return
		}

		ap.RecordPicAttempt(s.AccountId(), true, ipAddress, "")

		c, err := channel.NewProcessor(l, ctx).GetById(s.Channel())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve channel information being logged in to.")
			err = session.Announce(l)(ctx)(wp)(writer.ServerIP)(writer.ServerIPBodySimpleError(writer.ServerIPCodeServerUnderInspection))(s)
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
