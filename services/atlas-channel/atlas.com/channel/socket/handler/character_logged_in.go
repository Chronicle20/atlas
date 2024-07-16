package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterLoggedInHandle = "CharacterLoggedInHandle"

func CharacterLoggedInHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	setFieldFunc := session.Announce(wp)(writer.SetField)
	return func(s session.Model, r *request.Reader) {
		characterId := r.ReadUint32()
		buffer := r.GetRestAsBytes()
		l.Debugf("Handling login for character [%d]. buffer: %s", characterId, buffer)

		c, err := character.GetById(l, span, s.Tenant())(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] attempting to login.", characterId)
			session.Destroy(l, span, session.GetRegistry(), s.Tenant().Id)(s)
			return
		}

		s = session.SetAccountId(c.AccountId())(s.Tenant().Id, s.SessionId())
		s = session.SetCharacterId(c.Id())(s.Tenant().Id, s.SessionId())
		s = session.SetGm(c.Gm())(s.Tenant().Id, s.SessionId())

		resp, err := as.UpdateState(l, span, s.Tenant())(s.SessionId(), s.AccountId(), 3)
		if err != nil || resp.Code != "OK" {
			l.WithError(err).Errorf("Unable to update session for character [%d] attempting to switch to channel.", characterId)
			session.Destroy(l, span, session.GetRegistry(), s.Tenant().Id)(s)
			return
		}

		session.SessionCreated(l, span, s.Tenant())(s)

		l.Debugf("Writing SetField for character [%d].", c.Id())
		err = setFieldFunc(s, writer.SetFieldBody(l, s.Tenant())(s.ChannelId(), c))
		if err != nil {
			l.WithError(err).Errorf("Unable to show set field response for character [%d]", c.Id())
		}
	}
}
