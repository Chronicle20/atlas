package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"atlas-login/world/channel"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllSelectedPicHandle = "CharacterViewAllSelectedPicHandle"

func CharacterViewAllSelectedPicHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	serverIpFunc := session.Announce(l)(wp)(writer.ServerIP)
	return func(s session.Model, r *request.Reader) {
		pic := r.ReadAsciiString()
		characterId := r.ReadUint32()
		worldId := r.ReadUint32()
		macAddress := r.ReadAsciiString()
		macAddressWithHDDSerial := r.ReadAsciiString()
		l.Debugf("Character [%d] attempting to login via view all. worldId [%d], macAddress [%s], macAddressWithHDDSerial [%s], pic [%s].", characterId, worldId, macAddress, macAddressWithHDDSerial, pic)

		c, err := character.GetById(l, ctx)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d].", characterId)
			// TODO issue error
			return
		}

		t := s.Tenant()
		if c.WorldId() != byte(worldId) {
			l.Errorf("Character is not part of world provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		if c.AccountId() != s.AccountId() {
			l.Errorf("Character is not part of account provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		a, err := account.GetById(l, ctx)(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get account [%d].", s.AccountId())
			// TODO issue error
			return
		}

		if a.PIC() != pic {
			l.Errorf("Mismatch PIC between [%s] and [%s].", pic, a.PIC())
			// TODO issue error
			return
		}

		w, err := world.GetById(l, ctx)(byte(worldId))
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

		s = session.SetWorldId(byte(worldId))(t.Id(), s.SessionId())

		channel, err := channel.GetRandomInWorld(l, ctx)(byte(worldId))
		s = session.SetChannelId(channel.Id())(t.Id(), s.SessionId())

		resp, err := as.UpdateState(l, ctx)(s.SessionId(), s.AccountId(), 2)
		if err != nil || resp.Code != "OK" {
			l.WithError(err).Errorf("Unable to update session for character [%d] attempting to login.", characterId)
			err = serverIpFunc(s, writer.ServerIPBodySimpleError(l)(writer.ServerIPCodeTooManyConnectionRequests))
			if err != nil {
				l.WithError(err).Errorf("Unable to write server ip response due to error.")
				return
			}
			return
		}

		err = serverIpFunc(s, writer.ServerIPBody(l, t)(channel.IpAddress(), uint16(channel.Port()), characterId))
		if err != nil {
			l.WithError(err).Errorf("Unable to write server ip response due to error.")
			return
		}
	}
}
