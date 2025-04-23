package handler

import (
	"atlas-channel/chair"
	chair2 "atlas-channel/kafka/message/chair"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterChairInteractionHandle = "CharacterChairInteractionHandle"

func CharacterChairFixedHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	cp := chair.NewProcessor(l, ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		chairId := r.ReadInt16()
		if chairId == -1 {
			_ = cp.Cancel(s.Map(), s.CharacterId())
			return
		}

		_ = cp.Use(s.Map(), chair2.TypeFixed, uint32(chairId), s.CharacterId())
	}
}
