package handler

import (
	"atlas-login/account"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"atlas-login/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterListWorldHandle = "CharacterListWorldHandle"

func CharacterListWorldHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	serverStatusFunc := session.Announce(wp)(writer.ServerStatus)
	characterListFunc := session.Announce(wp)(writer.CharacterList)
	return func(s session.Model, r *request.Reader) {
		worldId := r.ReadByte()
		channelId := r.ReadByte()

		w, err := world.GetById(l, span)(worldId)
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

		s = session.SetWorldId(worldId)(s.SessionId())
		s = session.SetChannelId(channelId)(s.SessionId())

		a, err := account.GetById(l, span, s.Tenant())(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Cannot retrieve account")
			return
		}

		//cs, err := character.GetForWorld(l, span, s.Tenant())(s.AccountId(), worldId)
		//if err != nil {
		//	l.WithError(err).Errorf("Cannot retrieve account characters")
		//	return
		//}

		var cs []character.Model
		err = characterListFunc(s, writer.CharacterListBody(l, s.Tenant())(cs, worldId, 0, true, a.PIC(), int16(1), a.CharacterSlots()))
		if err != nil {
			l.WithError(err).Errorf("Unable to show character list")
		}
	}
}
