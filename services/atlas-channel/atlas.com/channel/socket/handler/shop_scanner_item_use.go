package handler

import (
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"

	character2 "atlas-channel/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	merchantsb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// ShopScannerItemUseHandleFunc handles the dedicated USE-inventory owl route
// (CWvsContext::SendShopScannerItemUseRequest, 231xxxx family double-clicked
// from the USE inventory). Validates the claimed item is a 231-family item
// actually present at the claimed slot before searching.
func ShopScannerItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := merchantsb.ShopScannerItemUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		itemId := item.Id(p.ItemId())
		if item.GetClassification(itemId) != item.ClassificationConsumableStoreSearch {
			l.Warnf("Character [%d] attempted shop scanner item use with non-scanner item [%d].", s.CharacterId(), itemId)
			return
		}

		a, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.TypeValueUse, p.Source())()
		if err != nil || item.Id(a.TemplateId()) != itemId {
			l.Warnf("Character [%d] attempted to use scanner item [%d] in slot [%d], but item not found or mismatched.", s.CharacterId(), itemId, p.Source())
			return
		}

		_ = shopscanner.NewProcessor(l, ctx).Search(wp)(s, p.SearchItemId(), p.Descending(), itemId, slot.Position(p.Source()), p.UpdateTime())
	}
}
