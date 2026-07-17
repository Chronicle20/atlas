package handler

import (
	character2 "atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/teleportrock"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	trsb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// itemInSlotFunc is a test seam for the inventory ownership check (package-var
// injection precedent: mystic_door_enter.go:25-51). Returns the template id of
// the USE-inventory item in the slot. GetItemInSlot returns an asset whose
// TemplateId() is compared — see character_cash_item_use.go:37.
var itemInSlotFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, slot int16) (uint32, error) {
	a, err := character2.NewProcessor(l, ctx).GetItemInSlot(characterId, inventory.TypeValueUse, slot)()
	if err != nil {
		return 0, err
	}
	return uint32(a.TemplateId()), nil
}

// useRockFunc is a test seam over the shared use-flow — handler tests in this
// package cannot reach the unexported seams inside atlas-channel/teleportrock,
// so they capture the invocation here instead. Shared with the cash branch
// (Task 19).
var useRockFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, target trpkt.Target) {
	return teleportrock.UseRock(l, ctx, wp)
}

// TeleportRockUseHandleFunc handles USE_TELEPORT_ROCK
// (CWvsContext::SendMapTransferItemUseRequest). Only the regular USE rock
// (232xxxx) arrives on this op — cash rocks ride CASH_ITEM_USE (design §1 Q1).
func TeleportRockUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := trsb.Use{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if !p.Valid() {
			// Client omitted the target payload (dialog closed without a
			// selection) — malformed by the client's own rules; drop silently.
			l.Warnf("Character [%d] sent USE_TELEPORT_ROCK without a target payload.", s.CharacterId())
			return
		}

		// Mirror the client guard: this op only carries 232xxxx.
		if p.ItemId()/10000 != 232 {
			l.Warnf("Character [%d] sent USE_TELEPORT_ROCK with non-rock item [%d].", s.CharacterId(), p.ItemId())
			return
		}

		// Verify the claimed slot actually holds the claimed item.
		templateId, err := itemInSlotFunc(l, ctx, s.CharacterId(), p.Slot())
		if err != nil || templateId != p.ItemId() {
			l.Warnf("Character [%d] attempted to use rock [%d] in slot [%d], but item not found or mismatched.", s.CharacterId(), p.ItemId(), p.Slot())
			return
		}

		useRockFunc(l, ctx, wp)(s, item.Id(p.ItemId()), p.Target())
	}
}
