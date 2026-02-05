package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterDistributeSpHandle = "CharacterDistributeSpHandle"

func CharacterDistributeSpHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		skillId := r.ReadUint32()
		_ = character.NewProcessor(l, ctx).RequestDistributeSp(s.Field(), s.CharacterId(), updateTime, skillId, 1)
	}
}
