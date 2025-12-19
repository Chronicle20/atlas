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

const RegisterPicHandle = "RegisterPicHandle"

func RegisterPicHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	t := tenant.MustFromContext(ctx)
	ap := account.NewProcessor(l, ctx)
	serverIpFunc := session.Announce(l)(wp)(writer.ServerIP)
	return func(s session.Model, r *request.Reader) {
		opt := r.ReadByte()
		characterId := r.ReadUint32()
		var sMacAddressWithHDDSerial = ""
		var sMacAddressWithHDDSerial2 = ""
		if t.Region() == "GMS" {
			sMacAddressWithHDDSerial = r.ReadAsciiString()
			sMacAddressWithHDDSerial2 = r.ReadAsciiString()
		}
		pic := r.ReadAsciiString()

		l.Debugf("Attempting to register PIC [%s]. opt [%d], character [%d], hwid [%s] hwid [%s].", pic, opt, characterId, sMacAddressWithHDDSerial, sMacAddressWithHDDSerial2)

		a, err := ap.GetById(s.AccountId())
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
		err = ap.UpdatePic(s.AccountId(), pic)
		if err != nil {
			l.WithError(err).Errorf("Unable to register PIC [%s] for account [%d].", pic, s.AccountId())
		}

		c, err := channel.NewProcessor(l, ctx).GetById(s.WorldId(), s.ChannelId())
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
