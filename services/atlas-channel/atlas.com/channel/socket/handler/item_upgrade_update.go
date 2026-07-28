package handler

import (
	"atlas-channel/consumable"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// ItemUpgradeUpdateHandleFunc handles the CUIItemUpgrade gauge-confirm packet.
// The client echoes the open-arm mode byte (returnResult) and the server's
// round-trip token (result), which packs hammerSlot|equipSlot. All
// authoritative validation happens in atlas-consumables against fresh state —
// a forged or replayed confirm is rejected there (design §4.1).
func ItemUpgradeUpdateHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := fieldsb.ItemUpgradeUpdate{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		hammerSlot, equipSlot := unpackViciousHammerToken(p.Result())
		err := consumable.NewProcessor(l, ctx).RequestViciousHammerUse(s.Field(), character.Id(s.CharacterId()), slot.Position(hammerSlot), slot.Position(equipSlot))
		if err != nil {
			l.WithError(err).Errorf("Character [%d] unable to request vicious hammer application.", s.CharacterId())
		}
	}
}
