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

func CharacterItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		slot := r.ReadInt16()
		itemId := r.ReadUint32()
		l.Debugf("Character [%d] using item [%d] from slot [%d]. updateTime [%d]", s.CharacterId(), itemId, slot, updateTime)
		_ = inventory.RequestItemConsume(l)(ctx)(s.CharacterId(), itemId, slot)
	}
}
