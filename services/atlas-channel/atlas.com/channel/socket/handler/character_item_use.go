package handler

import (
	"atlas-channel/character/inventory"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterItemUseHandle = "CharacterItemUseHandle"
const CharacterItemUseTownScrollHandle = "CharacterItemUseTownScrollHandle"

func CharacterItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		slot := r.ReadInt16()
		itemId := r.ReadUint32()
		_ = inventory.RequestItemConsume(l)(ctx)(s.CharacterId(), itemId, slot, updateTime)
	}
}

func CharacterItemUseTownScrollHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		slot := r.ReadInt16()
		itemId := r.ReadUint32()
		_ = inventory.RequestItemConsume(l)(ctx)(s.CharacterId(), itemId, slot, updateTime)
	}
}
