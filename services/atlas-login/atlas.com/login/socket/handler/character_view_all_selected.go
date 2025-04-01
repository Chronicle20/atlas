package handler

import (
	as "atlas-login/account/session"
	"atlas-login/character"
	"atlas-login/kafka/producer"
	"atlas-login/session"
	"atlas-login/socket/model"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"atlas-login/world/channel"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllSelectedHandle = "CharacterViewAllSelectedHandle"

func CharacterViewAllSelectedHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader) {
		characterId := r.ReadUint32()
		worldId := r.ReadUint32()
		macAddress := r.ReadAsciiString()
		macAddressWithHDDSerial := r.ReadAsciiString()
		l.Debugf("Character [%d] attempting to login via view all. worldId [%d], macAddress [%s], macAddressWithHDDSerial [%s].", characterId, worldId, macAddress, macAddressWithHDDSerial)

		c, err := character.GetById(l, ctx)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d].", characterId)
			// TODO issue error
			return
		}

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

		err = as.UpdateState(l, producer.ProviderImpl(l)(ctx))(s.SessionId(), s.AccountId(), 2, model.ChannelSelect{IPAddress: channel.IpAddress(), Port: uint16(channel.Port()), CharacterId: characterId})
		if err != nil {
			return
		}
	}
}
