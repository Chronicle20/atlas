package handler

import (
	"atlas-login/account"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"

	"github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterListWorldHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := login.WorldCharacterListRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		w, err := world.NewProcessor(l, ctx).GetById(p.WorldId())
		if err != nil {
			l.WithError(err).Errorf("Received a character list request for a world we do not have")
			return
		}

		if w.CapacityStatus() == world.StatusFull {
			err = session.Announce(l)(ctx)(wp)(writer.ServerStatus)(writer.ServerStatusBody(world.StatusFull))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to show that world %d is full", w.Id())
			}
			return
		}

		sp := session.NewProcessor(l, ctx)
		s = sp.SetWorldId(s.SessionId(), p.WorldId())
		s = sp.SetChannelId(s.SessionId(), p.ChannelId())

		a, err := account.NewProcessor(l, ctx).GetById(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Cannot retrieve account")
			return
		}
		cp := character.NewProcessor(l, ctx)
		cs, err := cp.GetForWorld(cp.InventoryDecorator())(s.AccountId(), w.Id())
		if err != nil {
			l.WithError(err).Errorf("Cannot retrieve account characters")
			return
		}

		err = session.Announce(l)(ctx)(wp)(writer.CharacterList)(writer.CharacterListBody(cs, p.WorldId(), 0, a.PIC(), int16(1), a.CharacterSlots()))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to show character list")
		}
	}
}
