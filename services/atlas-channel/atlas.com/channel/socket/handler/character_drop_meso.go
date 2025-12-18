package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterDropMesoHandle = "CharacterDropMesoHandle"

func CharacterDropMesoHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		amount := r.ReadUint32()
		l.Debugf("Character [%d] attempting to drop [%d] meso. updateTime [%d].", s.CharacterId(), amount, updateTime)
		_ = character.NewProcessor(l, ctx).RequestDropMeso(s.Map(), s.CharacterId(), amount)
	}
}
