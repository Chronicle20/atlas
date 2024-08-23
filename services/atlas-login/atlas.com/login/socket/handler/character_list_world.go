package handler

import (
	"atlas-login/account"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterListWorldHandle = "CharacterListWorldHandle"

func CharacterListWorldHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	serverStatusFunc := session.Announce(l)(wp)(writer.ServerStatus)
	characterListFunc := session.Announce(l)(wp)(writer.CharacterList)
	return func(s session.Model, r *request.Reader) {
		var gameStartMode = byte(0)
		if s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 28 {
			// GMS v28 is not definite here, but this is not present in 28.
			gameStartMode = r.ReadByte()
		}
		worldId := r.ReadByte()
		channelId := r.ReadByte()

		var socketAddr int32
		if s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 12 {
			socketAddr = r.ReadInt32()
		} else if s.Tenant().Region == "JMS" {
			socketAddr = r.ReadInt32()
		}

		l.Debugf("Handling [CharacterListWorld]. gameStartMode=[%d], worldId=[%d], channelId=[%d], socketAddr=[%d]", gameStartMode, worldId, channelId, socketAddr)

		w, err := world.GetById(l, ctx, s.Tenant())(worldId)
		if err != nil {
			l.WithError(err).Errorf("Received a character list request for a world we do not have")
			return
		}

		if w.CapacityStatus() == world.StatusFull {
			err = serverStatusFunc(s, writer.ServerStatusBody(l)(world.StatusFull))
			if err != nil {
				l.WithError(err).Errorf("Unable to show that world %d is full", w.Id())
			}
			return
		}

		s = session.SetWorldId(worldId)(s.Tenant().Id, s.SessionId())
		s = session.SetChannelId(channelId)(s.Tenant().Id, s.SessionId())

		a, err := account.GetById(l, ctx, s.Tenant())(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Cannot retrieve account")
			return
		}

		cs, err := character.GetForWorld(l, ctx, s.Tenant())(s.AccountId(), worldId)
		if err != nil {
			l.WithError(err).Errorf("Cannot retrieve account characters")
			return
		}

		err = characterListFunc(s, writer.CharacterListBody(l, s.Tenant())(cs, worldId, 0, a.PIC(), int16(1), a.CharacterSlots()))
		if err != nil {
			l.WithError(err).Errorf("Unable to show character list")
		}
	}
}
