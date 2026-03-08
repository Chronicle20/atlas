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

	"github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterViewAllSelectedHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := login.AllCharacterListSelect{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		c, err := character.NewProcessor(l, ctx).GetById(character.NewProcessor(l, ctx).InventoryDecorator())(p.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d].", p.CharacterId())
			// TODO issue error
			return
		}

		if c.WorldId() != p.WorldId() {
			l.Errorf("Character is not part of world provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		if c.AccountId() != s.AccountId() {
			l.Errorf("Character is not part of account provided by client. Potential packet exploit from [%d]. Terminating session.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		w, err := world.NewProcessor(l, ctx).GetById(p.WorldId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get world [%d].", p.WorldId())
			// TODO issue error
			return
		}

		if w.CapacityStatus() == world.StatusFull {
			l.Errorf("World [%d] has capacity status [%d].", p.WorldId(), w.CapacityStatus())
			// TODO issue error
			return
		}

		s = session.NewProcessor(l, ctx).SetWorldId(s.SessionId(), p.WorldId())

		ch, err := channel.NewProcessor(l, ctx).GetRandomInWorld(p.WorldId())
		s = session.NewProcessor(l, ctx).SetChannelId(s.SessionId(), ch.ChannelId())

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelSelect{IPAddress: ch.IpAddress(), Port: uint16(ch.Port()), CharacterId: p.CharacterId()})
		if err != nil {
			return
		}
	}
}
