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

const CharacterChairPortableHandle = "CharacterChairPortableHandle"

func CharacterChairPortableHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		itemId := r.ReadUint32()
		_ = chair.NewProcessor(l, ctx).Use(s.Field(), chair2.TypePortable, itemId, s.CharacterId())
	}
}
