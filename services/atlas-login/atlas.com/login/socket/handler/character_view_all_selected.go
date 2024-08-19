package handler

import (
	as "atlas-login/account/session"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"atlas-login/world/channel"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllSelectedHandle = "CharacterViewAllSelectedHandle"

func CharacterViewAllSelectedHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	serverIpFunc := session.Announce(l)(wp)(writer.ServerIP)
	return func(s session.Model, r *request.Reader) {
		characterId := r.ReadUint32()
		worldId := r.ReadUint32()
		macAddress := r.ReadAsciiString()
		macAddressWithHDDSerial := r.ReadAsciiString()
		l.Debugf("Character [%d] attempting to login via view all. worldId [%d], macAddress [%s], macAddressWithHDDSerial [%s].", characterId, worldId, macAddress, macAddressWithHDDSerial)

		c, err := character.GetById(l, span, s.Tenant())(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d].", characterId)
			// TODO issue error
			return
		}

		if c.WorldId() != byte(worldId) {
			l.Errorf("Character is not part of world provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			session.Destroy(l, span, session.GetRegistry(), s.Tenant().Id)(s)
			return
		}

		if c.AccountId() != s.AccountId() {
			l.Errorf("Character is not part of account provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			session.Destroy(l, span, session.GetRegistry(), s.Tenant().Id)(s)
			return
		}

		w, err := world.GetById(l, span, s.Tenant())(byte(worldId))
		if err != nil {
			l.WithError(err).Errorf("Unable to get world [%d].", worldId)
			// TODO issue error
			return
		}

		if w.CapacityStatus() == world.StatusFull {
			l.Errorf("World [%d] has capacity status [%d].", worldId, w.CapacityStatus())
			// TODO issue error
			return
		}

		s = session.SetWorldId(byte(worldId))(s.Tenant().Id, s.SessionId())

		channel, err := channel.GetRandomInWorld(l, span, s.Tenant())(byte(worldId))
		s = session.SetChannelId(channel.Id())(s.Tenant().Id, s.SessionId())

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

		err = serverIpFunc(s, writer.ServerIPBody(l, s.Tenant())(channel.IpAddress(), uint16(channel.Port()), characterId))
		if err != nil {
			l.WithError(err).Errorf("Unable to write server ip response due to error.")
			return
		}
	}
}
