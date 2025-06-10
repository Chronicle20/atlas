package handler

import (
	"atlas-channel/consumable"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	CharacterItemUseHandle           = "CharacterItemUseHandle"
	CharacterItemUseTownScrollHandle = "CharacterItemUseTownScrollHandle"
	CharacterItemUseScrollHandle     = "CharacterItemUseScrollHandle"
	CharacterItemUseSummonBagHandle  = "CharacterItemUseSummonBagHandle"
)

func CharacterItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		slot := r.ReadInt16()
		itemId := r.ReadUint32()
		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.CharacterId(), itemId, slot, updateTime)
	}
}

func CharacterItemUseTownScrollHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		slot := r.ReadInt16()
		itemId := r.ReadUint32()
		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.CharacterId(), itemId, slot, updateTime)
	}
}

func CharacterItemUseScrollHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		scrollSlot := r.ReadInt16()
		equipSlot := r.ReadInt16()
		bWhiteScroll := r.ReadInt16()
		whiteScroll := (bWhiteScroll & 2) == 2
		legendarySpirit := r.ReadBool()
		_ = consumable.NewProcessor(l, ctx).RequestScrollUse(s.CharacterId(), scrollSlot, equipSlot, whiteScroll, legendarySpirit, updateTime)
	}
}

func CharacterItemUseSummonBagHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		slot := r.ReadInt16()
		itemId := r.ReadUint32()
		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.CharacterId(), itemId, slot, updateTime)
	}
}
