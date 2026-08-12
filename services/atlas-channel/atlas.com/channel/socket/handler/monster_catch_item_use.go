package handler

import (
	"atlas-channel/consumable"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	monstersb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// MonsterCatchItemUseHandleFunc decodes USE_CATCH_ITEM and forwards it. It
// performs no validation and holds no state: the item checks belong to
// atlas-consumables and the monster checks to atlas-monsters, which is
// authoritative and fail-closed. The opcode arrives from tenant configuration
// via the template route — never a constant here (DOM-25).
func MonsterCatchItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var p monstersb.UseCatchItem
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = consumable.NewProcessor(l, ctx).RequestCatchMonster(s.Field(), character.Id(s.CharacterId()), item.Id(p.ItemId()), slot.Position(p.Slot()), p.MonsterUniqueId())
	}
}
