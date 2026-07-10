package cart

import (
	mtslisting "atlas-channel/mts/listing"
	mtswish "atlas-channel/mts/wish"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/sirupsen/logrus"
)

// Items renders a character's Cart (SET_ZZIM favorites) as browse ITCITEMs, one
// per cart entry, by resolving each entry's favorited LISTING (its stored
// listingSerial) to the live listing and rendering THAT listing via
// mtslisting.ToMtsItem. The cart entry records the exact listing the player
// favorited, so the row shows that listing's all-in price (nPrice +
// nContractFee), its "Sold Until" date, and its serial as the nITCSN — and
// BUY_ZZIM / DELETE_ZZIM address that same listing.
//
// This replaces the old template re-resolution (BestActiveListing): storing only
// the item template and rendering "the cheapest active listing by another seller"
// showed a DIFFERENT listing's price and sold-until date and could resolve to a
// listing the player never favorited (or exclude their own) — task-102 live
// finding (carted a sword listed at 11200, cart showed an unrelated 2151 row).
//
// A cart entry whose favorited listing no longer exists (sold, expired, or
// cancelled) is skipped: it drops off the Cart until the item is favorited again.
// Legacy cart entries created before listingSerial was tracked carry serial 0,
// which never matches a real listing (serials start at 1), so they drop too.
func Items(l logrus.FieldLogger, ctx context.Context, worldId world.Id, characterId uint32) ([]fieldcb.MtsItem, error) {
	ws, err := mtswish.NewProcessor(l, ctx).GetByCharacterAndType(characterId, mtswish.TypeCart)
	if err != nil {
		l.WithError(err).Errorf("Unable to load cart entries for character [%d]; rendering empty cart.", characterId)
		return nil, err
	}

	// Resolve ALL favorited listings in ONE browse (serial IN ...) rather than a
	// GetBySerial per entry: the cart re-renders on every wish add/remove and every
	// purchase, so a per-entry fan-out would be N atlas-mts round-trips each time.
	serials := make([]uint32, 0, len(ws))
	for _, w := range ws {
		if w.ListingSerial() != 0 {
			serials = append(serials, w.ListingSerial())
		}
	}
	byNITCSN := make(map[uint32]mtslisting.Model, len(serials))
	if len(serials) > 0 {
		ms, berr := mtslisting.NewProcessor(l, ctx).Browse(worldId, mtslisting.BrowseFilter{Serials: serials, PageSize: -1})
		if berr != nil {
			l.WithError(berr).Errorf("Unable to resolve cart listings for character [%d]; rendering empty cart.", characterId)
			return nil, berr
		}
		for _, m := range ms {
			byNITCSN[m.ItcSn()] = m
		}
	}

	// Render in cart order. A favorited listing missing from the browse is gone
	// (sold / expired / cancelled — the browse returns only active listings) or is a
	// legacy serial-0 entry; either way it drops off the cart until re-favorited.
	items := make([]fieldcb.MtsItem, 0, len(ws))
	for _, w := range ws {
		lm, ok := byNITCSN[w.ListingSerial()]
		if !ok {
			continue
		}
		items = append(items, mtslisting.ToMtsItem(lm))
	}
	return items, nil
}
