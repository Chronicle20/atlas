package handler

import (
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
		_ = r.ReadAsciiString() // pic - not logged for security
		characterId := r.ReadUint32()

		if t.Region() == "GMS" {
			_ = r.ReadAsciiString() // sMacAddressWithHDDSerial
			_ = r.ReadAsciiString() // sMacAddressWithHDDSerial2
		}
		l.Debugf("Character [%d] selected for login to channel [%d:%d].", characterId, s.WorldId(), s.ChannelId())
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
