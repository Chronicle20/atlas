package handler

import (
	"atlas-channel/chair"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterChairInteractionHandle = "CharacterChairInteractionHandle"

func CharacterChairFixedHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		chairId := r.ReadInt16()
		if chairId == -1 {
			_ = chair.Cancel(l)(ctx)(s.Map(), s.CharacterId())
			return
		}

		_ = chair.Use(l)(ctx)(s.Map(), chair.ChairTypeFixed, uint32(chairId), s.CharacterId())
	}
}
