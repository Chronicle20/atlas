package cart

import (
	mtslisting "atlas-channel/mts/listing"
	mtswish "atlas-channel/mts/wish"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/sirupsen/logrus"
)

// itcSaleTypeFixed mirrors listing.SaleType's wire string for a fixed-price sale.
// The Cart resolves each favorited item to a FIXED listing only: BUY_ZZIM settles
// at listValue (BuyNow=false), which is correct for a fixed sale but wrong for an
// auction (an auction is transacted via bids / buy-now, not Zzim), so auctions are
// intentionally excluded from the Cart's buyable rendering.
const itcSaleTypeFixed = "fixed"

// Items renders a character's Cart (SET_ZZIM favorites) as browse ITCITEMs, one
// per cart entry, by resolving each favorited item to its current best (cheapest,
// not-self, active) FIXED listing and rendering THAT listing via
// mtslisting.ToMtsItem. This is the fix for two coupled Cart bugs:
//
//   - the row shows the all-in price (nPrice + nContractFee) because ToMtsItem
//     stamps the marked-up contract fee — a bare cart wish carried no fee and so
//     rendered the base price;
//   - the row's nITCSN is the LISTING serial, so BUY_ZZIM / DELETE_ZZIM address a
//     real listing. Rendering the wish's own serial made BUY_ZZIM miss GetBySerial
//     server-side and fail as ITEM_SOLD ("the item has been sold" — task-102 live
//     finding), since wishes and listings share one serial space and a wish serial
//     never matches a listing row.
//
// A cart entry whose item has no active listing the viewer can buy (sold out, or
// every listing is the viewer's own) is skipped: a favorited item that is no
// longer for sale simply drops off the Cart until it is listed again.
func Items(l logrus.FieldLogger, ctx context.Context, worldId world.Id, characterId uint32) ([]fieldcb.MtsItem, error) {
	ws, err := mtswish.NewProcessor(l, ctx).GetByCharacterAndType(characterId, mtswish.TypeCart)
	if err != nil {
		l.WithError(err).Errorf("Unable to load cart entries for character [%d]; rendering empty cart.", characterId)
		return nil, err
	}
	items := make([]fieldcb.MtsItem, 0, len(ws))
	for _, w := range ws {
		lm, ok := BestActiveListing(l, ctx, worldId, characterId, w.ItemId())
		if !ok {
			continue
		}
		items = append(items, mtslisting.ToMtsItem(lm))
	}
	return items, nil
}

// BestActiveListing returns the cheapest active FIXED listing of itemId in the
// world that the viewer does not own, or ok=false when none exists. It is the
// listing a Cart entry for that item resolves to — for rendering and for the
// BUY_ZZIM / DELETE_ZZIM serial. "Cheapest by listValue" is deterministic; the
// per-(character, item) cart model cannot distinguish two listings of the same
// item, so the best available deal is the faithful resolution.
func BestActiveListing(l logrus.FieldLogger, ctx context.Context, worldId world.Id, viewerId uint32, itemId uint32) (mtslisting.Model, bool) {
	ms, err := mtslisting.NewProcessor(l, ctx).Browse(worldId, mtslisting.BrowseFilter{
		TemplateIds:     []uint32{itemId},
		ExcludeSellerId: viewerId,
		SaleType:        itcSaleTypeFixed,
		PageSize:        -1,
	})
	if err != nil || len(ms) == 0 {
		return mtslisting.Model{}, false
	}
	best := ms[0]
	for _, m := range ms[1:] {
		if m.ListValue() < best.ListValue() {
			best = m
		}
	}
	return best, true
}
