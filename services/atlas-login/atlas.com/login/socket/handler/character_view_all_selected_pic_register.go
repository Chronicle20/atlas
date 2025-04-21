package handler

import (
	"atlas-login/account"
	as "atlas-login/account/session"
	"atlas-login/channel"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/model"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllSelectedPicRegisterHandle = "CharacterViewAllSelectedPicRegisterHandle"

func CharacterViewAllSelectedPicRegisterHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader) {
	t := tenant.MustFromContext(ctx)
	cp := character.NewProcessor(l, ctx)
	return func(s session.Model, r *request.Reader) {
		opt := r.ReadByte()
		characterId := r.ReadUint32()
		worldId := r.ReadUint32()
		macAddress := r.ReadAsciiString()
		macAddressWithHDDSerial := r.ReadAsciiString()
		pic := r.ReadAsciiString()
		l.Debugf("Character [%d] attempting to login via view all. opt [%d], worldId [%d], macAddress [%s], macAddressWithHDDSerial [%s], pic [%s].", characterId, opt, worldId, macAddress, macAddressWithHDDSerial, pic)

		c, err := cp.GetById(cp.InventoryDecorator())(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d].", characterId)
			// TODO issue error
			return
		}

		if c.WorldId() != byte(worldId) {
			l.Errorf("Character is not part of world provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		if c.AccountId() != s.AccountId() {
			l.Errorf("Character is not part of account provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		err = account.NewProcessor(l, ctx).UpdatePic(s.AccountId(), pic)
		if err != nil {
			l.WithError(err).Errorf("Unable to PIC for account [%d].", s.AccountId())
			// TODO issue error
			return
		}

		w, err := world.NewProcessor(l, ctx).GetById(byte(worldId))
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

		channel, err := channel.NewProcessor(l, ctx).GetRandomInWorld(byte(worldId))
		s = session.SetChannelId(channel.ChannelId())(t.Id(), s.SessionId())

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelSelect{IPAddress: channel.IpAddress(), Port: uint16(channel.Port()), CharacterId: characterId})
		if err != nil {
			return
		}
	}
}
