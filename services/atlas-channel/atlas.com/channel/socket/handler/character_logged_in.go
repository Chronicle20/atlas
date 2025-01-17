package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/character"
	"atlas-channel/kafka/producer"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterLoggedInHandle = "CharacterLoggedInHandle"

func CharacterLoggedInHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, _ map[string]interface{}) {
	return func(s session.Model, r *request.Reader, _ map[string]interface{}) {
		characterId := r.ReadUint32()
		buffer := r.GetRestAsBytes()
		l.Debugf("Handling login for character [%d]. buffer: %s", characterId, buffer)

		c, err := character.GetById(l)(ctx)(characterId)
		if err != nil {
			return
		}

		err = as.UpdateState(l, producer.ProviderImpl(l)(ctx))(s.SessionId(), c.AccountId(), 1, model2.SetField{CharacterId: characterId})
		if err != nil {
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
		}
		return

	}
}
