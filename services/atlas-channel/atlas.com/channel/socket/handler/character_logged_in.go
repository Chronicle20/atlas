package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/character/key"
	"atlas-channel/kafka/producer"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterLoggedInHandle = "CharacterLoggedInHandle"

func CharacterLoggedInHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, _ map[string]interface{}) {
	t := tenant.MustFromContext(ctx)

	setFieldFunc := session.Announce(l)(ctx)(wp)(writer.SetField)
	characterKeyMapFunc := session.Announce(l)(ctx)(wp)(writer.CharacterKeyMap)
	return func(s session.Model, r *request.Reader, _ map[string]interface{}) {
		characterId := r.ReadUint32()
		buffer := r.GetRestAsBytes()
		l.Debugf("Handling login for character [%d]. buffer: %s", characterId, buffer)

		c, err := character.GetByIdWithInventory(l)(ctx)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] attempting to login.", characterId)
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}
		bl, err := buddylist.GetById(l)(ctx)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate buddylist [%d] attempting to login.", characterId)
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		s = session.SetAccountId(c.AccountId())(t.Id(), s.SessionId())
		s = session.SetCharacterId(c.Id())(t.Id(), s.SessionId())
		s = session.SetGm(c.Gm())(t.Id(), s.SessionId())
		s = session.SetMapId(c.MapId())(t.Id(), s.SessionId())

		resp, err := as.UpdateState(l)(ctx)(s.SessionId(), s.AccountId(), 1)
		if err != nil || resp.Code != "OK" {
			l.WithError(err).Errorf("Unable to update session for character [%d] attempting to switch to channel.", characterId)
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		session.EmitCreated(producer.ProviderImpl(l)(ctx))(s)

		go func() {
			l.Debugf("Writing SetField for character [%d].", c.Id())
			err = setFieldFunc(s, writer.SetFieldBody(l, t)(s.ChannelId(), c, bl))
			if err != nil {
				l.WithError(err).Errorf("Unable to show set field response for character [%d]", c.Id())
			}
		}()
		go func() {
			km, err := model.CollectToMap[key.Model, int32, key.Model](key.ByCharacterIdProvider(l)(ctx)(characterId), func(m key.Model) int32 {
				return m.Key()
			}, func(m key.Model) key.Model {
				return m
			})()
			if err != nil {
				l.WithError(err).Errorf("Unable to show key map for character [%d].", characterId)
				return
			}

			err = characterKeyMapFunc(s, writer.CharacterKeyMapBody(km))
			if err != nil {
				l.WithError(err).Errorf("Unable to show key map for character [%d].", characterId)
			}
		}()
	}
}
