package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/character"
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

		c, err := character.NewProcessor(l, ctx).GetById()(characterId)
		if err != nil {
			return
		}

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), c.AccountId(), 1, model2.SetField{CharacterId: characterId})
		if err != nil {
			_ = session.NewProcessor(l, ctx).Destroy(s)
		}
		return

	}
}
