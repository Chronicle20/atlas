package handler

import (
	"atlas-login/account"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"atlas-login/world/channel"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllSelectedPicRegisterHandle = "CharacterViewAllSelectedPicRegisterHandle"

func CharacterViewAllSelectedPicRegisterHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	serverIpFunc := session.Announce(l)(wp)(writer.ServerIP)
	return func(s session.Model, r *request.Reader) {
		opt := r.ReadByte()
		characterId := r.ReadUint32()
		worldId := r.ReadUint32()
		macAddress := r.ReadAsciiString()
		macAddressWithHDDSerial := r.ReadAsciiString()
		pic := r.ReadAsciiString()
		l.Debugf("Character [%d] attempting to login via view all. opt [%d], worldId [%d], macAddress [%s], macAddressWithHDDSerial [%s], pic [%s].", characterId, opt, worldId, macAddress, macAddressWithHDDSerial, pic)

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

		err = account.UpdatePic(l, span, s.Tenant())(s.AccountId(), pic)
		if err != nil {
			l.WithError(err).Errorf("Unable to PIC for account [%d].", s.AccountId())
			// TODO issue error
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

		err = serverIpFunc(s, writer.ServerIPBody(l, s.Tenant())(channel.IpAddress(), uint16(channel.Port()), characterId))
		if err != nil {
			l.WithError(err).Errorf("Unable to write server ip response due to error.")
			return
		}
	}
}
