package handler

import (
	"atlas-channel/consumable"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const PetItemUseHandle = "PetItemUseHandle"

func PetItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		petId := r.ReadUint64()
		buffSkill := r.ReadBool()
		updateTime := r.ReadUint32()
		source := slot.Position(r.ReadInt16())
		itemId := item.Id(r.ReadUint32())
		l.Debugf("Character [%d] pet [%d] attempting to use item [%d] from slot [%d]. updateTime [%d], buffSkill [%t].", s.CharacterId(), petId, itemId, source, updateTime, buffSkill)
		_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), itemId, source, updateTime)
	}
}
