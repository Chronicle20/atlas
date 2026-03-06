package handler

import (
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

const CharacterViewAllSelectedHandle = "CharacterViewAllSelectedHandle"

func CharacterViewAllSelectedHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		characterId := r.ReadUint32()
		worldId := world2.Id(r.ReadUint32())
		macAddress := r.ReadAsciiString()
		macAddressWithHDDSerial := r.ReadAsciiString()
		l.Debugf("Character [%d] attempting to login via view all. worldId [%d], macAddress [%s], macAddressWithHDDSerial [%s].", characterId, worldId, macAddress, macAddressWithHDDSerial)

		c, err := character.NewProcessor(l, ctx).GetById(character.NewProcessor(l, ctx).InventoryDecorator())(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d].", characterId)
			// TODO issue error
			return
		}

		if c.WorldId() != worldId {
			l.Errorf("Character is not part of world provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		if c.AccountId() != s.AccountId() {
			l.Errorf("Character is not part of account provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
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

		s = session.NewProcessor(l, ctx).SetWorldId(s.SessionId(), worldId)

		ch, err := channel.NewProcessor(l, ctx).GetRandomInWorld(worldId)
		s = session.NewProcessor(l, ctx).SetChannelId(s.SessionId(), ch.ChannelId())

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelSelect{IPAddress: ch.IpAddress(), Port: uint16(ch.Port()), CharacterId: characterId})
		if err != nil {
			return
		}
	}
}
