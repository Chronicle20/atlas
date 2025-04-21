package handler

import (
	"atlas-login/account"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterListWorldHandle = "CharacterListWorldHandle"

func CharacterListWorldHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	t := tenant.MustFromContext(ctx)
	serverStatusFunc := session.Announce(l)(wp)(writer.ServerStatus)
	characterListFunc := session.Announce(l)(wp)(writer.CharacterList)
	return func(s session.Model, r *request.Reader) {
		var gameStartMode = byte(0)

		if t.Region() == "GMS" && t.MajorVersion() > 28 {
			// GMS v28 is not definite here, but this is not present in 28.
			gameStartMode = r.ReadByte()
		}
		worldId := r.ReadByte()
		channelId := r.ReadByte()

		var socketAddr int32
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			socketAddr = r.ReadInt32()
		} else if t.Region() == "JMS" {
			socketAddr = r.ReadInt32()
		}

		l.Debugf("Handling [CharacterListWorld]. gameStartMode=[%d], worldId=[%d], channelId=[%d], socketAddr=[%d]", gameStartMode, worldId, channelId, socketAddr)

		w, err := world.NewProcessor(l, ctx).GetById(worldId)
		if err != nil {
			l.WithError(err).Errorf("Received a character list request for a world we do not have")
			return
		}

		if w.CapacityStatus() == world.StatusFull {
			err = serverStatusFunc(s, writer.ServerStatusBody(world.StatusFull))
			if err != nil {
				l.WithError(err).Errorf("Unable to show that world %d is full", w.Id())
			}
			return
		}

		sp := session.NewProcessor(l, ctx)
		s = sp.SetWorldId(s.SessionId(), worldId)
		s = sp.SetChannelId(s.SessionId(), channelId)

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

		err = characterListFunc(s, writer.CharacterListBody(t)(cs, worldId, 0, a.PIC(), int16(1), a.CharacterSlots()))
		if err != nil {
			l.WithError(err).Errorf("Unable to show character list")
		}
	}
}
