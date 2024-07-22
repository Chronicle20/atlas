package handler

import (
	as "atlas-login/account/session"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world/channel"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterSelectedPicHandle = "CharacterSelectedPicHandle"

func CharacterSelectedPicHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	serverIpFunc := session.Announce(l)(wp)(writer.ServerIP)
	return func(s session.Model, r *request.Reader) {
		pic := r.ReadAsciiString()
		characterId := r.ReadUint32()
		var sMacAddressWithHDDSerial = ""
		var sMacAddressWithHDDSerial2 = ""
		if s.Tenant().Region == "GMS" {
			sMacAddressWithHDDSerial = r.ReadAsciiString()
			sMacAddressWithHDDSerial2 = r.ReadAsciiString()
		}
		l.Debugf("Character [%d] selected for login to channel [%d:%d]. pic [%s] hwid [%s] hwid [%s].", characterId, s.WorldId(), s.ChannelId(), pic, sMacAddressWithHDDSerial, sMacAddressWithHDDSerial2)
		c, err := channel.GetById(l, span, s.Tenant())(s.WorldId(), s.ChannelId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve channel information being logged in to.")
			err = serverIpFunc(s, writer.ServerIPBodySimpleError(l)(writer.ServerIPCodeServerUnderInspection))
			if err != nil {
				l.WithError(err).Errorf("Unable to write server ip response due to error.")
				return
			}
			return
		}

		resp, err := as.UpdateState(l, span, s.Tenant())(s.SessionId(), s.AccountId(), 2)
		if err != nil || resp.Code != "OK" {
			l.WithError(err).Errorf("Unable to update session for character [%d] attempting to login.", characterId)
			err = serverIpFunc(s, writer.ServerIPBodySimpleError(l)(writer.ServerIPCodeTooManyConnectionRequests))
			if err != nil {
				l.WithError(err).Errorf("Unable to write server ip response due to error.")
				return
			}
			return
		}

		err = serverIpFunc(s, writer.ServerIPBody(l, s.Tenant())(c.IpAddress(), uint16(c.Port()), characterId))
		if err != nil {
			l.WithError(err).Errorf("Unable to write server ip response due to error.")
			return
		}
	}
}
