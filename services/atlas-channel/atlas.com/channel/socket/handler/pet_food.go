package handler

import (
	"atlas-channel/character/inventory"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const PetFoodHandle = "PetFoodHandle"

func PetFoodHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		slot := r.ReadInt16()
		itemId := r.ReadUint32()
		l.Debugf("Character [%d] attempting to use pet food item [%d]. updateTime [%d], slot [%d]", s.CharacterId(), itemId, updateTime, slot)
		_ = inventory.RequestItemConsume(l)(ctx)(s.CharacterId(), itemId, slot)
	}
}
