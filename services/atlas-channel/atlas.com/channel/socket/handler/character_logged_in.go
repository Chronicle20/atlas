package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/character"
	"atlas-channel/kafka/producer"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterLoggedInHandle = "CharacterLoggedInHandle"

func CharacterLoggedInHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, _ map[string]interface{}) {
	setFieldFunc := session.Announce(l)(wp)(writer.SetField)
	return func(s session.Model, r *request.Reader, _ map[string]interface{}) {
		characterId := r.ReadUint32()
		buffer := r.GetRestAsBytes()
		l.Debugf("Handling login for character [%d]. buffer: %s", characterId, buffer)

		c, err := character.GetByIdWithInventory(l, ctx, s.Tenant())(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] attempting to login.", characterId)
			session.Destroy(l, ctx, session.GetRegistry(), s.Tenant().Id)(s)
			return
		}

		s = session.SetAccountId(c.AccountId())(s.Tenant().Id, s.SessionId())
		s = session.SetCharacterId(c.Id())(s.Tenant().Id, s.SessionId())
		s = session.SetGm(c.Gm())(s.Tenant().Id, s.SessionId())
		s = session.SetMapId(c.MapId())(s.Tenant().Id, s.SessionId())

		resp, err := as.UpdateState(l, ctx, s.Tenant())(s.SessionId(), s.AccountId(), 1)
		if err != nil || resp.Code != "OK" {
			l.WithError(err).Errorf("Unable to update session for character [%d] attempting to switch to channel.", characterId)
			session.Destroy(l, ctx, session.GetRegistry(), s.Tenant().Id)(s)
			return
		}

		session.EmitCreated(producer.ProviderImpl(l)(ctx), s.Tenant())(s)

		l.Debugf("Writing SetField for character [%d].", c.Id())
		err = setFieldFunc(s, writer.SetFieldBody(l, s.Tenant())(s.ChannelId(), c))
		if err != nil {
			l.WithError(err).Errorf("Unable to show set field response for character [%d]", c.Id())
		}
	}
}
