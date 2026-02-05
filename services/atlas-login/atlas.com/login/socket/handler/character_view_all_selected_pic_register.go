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
	world2 "github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllSelectedPicRegisterHandle = "CharacterViewAllSelectedPicRegisterHandle"

func CharacterViewAllSelectedPicRegisterHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader) {
	cp := character.NewProcessor(l, ctx)
	sp := session.NewProcessor(l, ctx)
	return func(s session.Model, r *request.Reader) {
		opt := r.ReadByte()
		characterId := r.ReadUint32()
		worldId := world2.Id(r.ReadUint32())
		_ = r.ReadAsciiString() // macAddress - not logged for security
		_ = r.ReadAsciiString() // macAddressWithHDDSerial - not logged for security
		pic := r.ReadAsciiString()
		l.Debugf("Character [%d] attempting to login via view all. opt [%d], worldId [%d].", characterId, opt, worldId)

		c, err := cp.GetById(cp.InventoryDecorator())(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d].", characterId)
			// TODO issue error
			return
		}

		if c.WorldId() != worldId {
			l.Errorf("Character is not part of world provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			_ = sp.Destroy(s)
			return
		}

		if c.AccountId() != s.AccountId() {
			l.Errorf("Character is not part of account provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			_ = sp.Destroy(s)
			return
		}

		err = account.NewProcessor(l, ctx).UpdatePic(s.AccountId(), pic)
		if err != nil {
			l.WithError(err).Errorf("Unable to PIC for account [%d].", s.AccountId())
			// TODO issue error
			return
		}

		w, err := world.NewProcessor(l, ctx).GetById(worldId)
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

		s = sp.SetWorldId(s.SessionId(), worldId)

		ch, err := channel.NewProcessor(l, ctx).GetRandomInWorld(worldId)
		s = sp.SetChannelId(s.SessionId(), ch.ChannelId())

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelSelect{IPAddress: ch.IpAddress(), Port: uint16(ch.Port()), CharacterId: characterId})
		if err != nil {
			return
		}
	}
}
