package wanted

import (
	"atlas-channel/character"
	mtswish "atlas-channel/mts/wish"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/sirupsen/logrus"
)

// WorldItems renders the cross-character Wanted tab (ITC section 2): every want-ad
// in the world EXCEPT the viewer's own, each carrying the owner's display name in
// the seller column. The Wanted tab shows other players' buy-orders you can fulfill
// (the seller counterpart to the For Sale tab, which excludes your own listings);
// your own want-ads appear under My Page -> Offers. A single shared renderer is
// used by BOTH the synchronous browse arm and the post-mutation re-push
// (announceWishList) so the two never diverge — the re-push previously rendered the
// viewer's OWN want-ads here, so a poster saw their own ad in the Wanted tab after
// posting/cancelling (task-102 live finding).
// categorySub is the item sub-tab (0=all, 1=equip, 2=use, 3=setup, 4=etc, 5=cash):
// the Wanted tab carries the same item-type sub-tabs as every other browse, so a
// want-ad is included only when its item's inventory type matches (0 shows all).
// Mirrors wishItems / the public browse's subCategory filter — without it the
// Wanted sub-tabs showed every want-ad regardless of type (task-102 finding).
func WorldItems(l logrus.FieldLogger, ctx context.Context, worldId world.Id, viewerId uint32, categorySub uint32) ([]fieldcb.MtsItem, error) {
	ws, err := mtswish.NewProcessor(l, ctx).GetWantedByWorld(byte(worldId))
	if err != nil {
		l.WithError(err).Errorf("Unable to load world want-ads for world [%d]; rendering empty Wanted list.", byte(worldId))
		return nil, err
	}
	items := make([]fieldcb.MtsItem, 0, len(ws))
	for _, w := range ws {
		if w.CharacterId() == viewerId {
			continue
		}
		if categorySub != 0 {
			if it, ok := inventory.TypeFromItemId(item.Id(w.ItemId())); !ok || uint32(it) != categorySub {
				continue
			}
		}
		items = append(items, toWantAdItem(l, ctx, w))
	}
	return items, nil
}

// toWantAdItem maps one cross-character want-ad to an ITCITEM, resolving the
// owner's display name into the sGameID column. The name lookup is best-effort: a
// failure yields "" (a blank seller column) and the want-ad is STILL included — a
// want-ad must never be dropped because its owner's name could not be resolved.
func toWantAdItem(l logrus.FieldLogger, ctx context.Context, w mtswish.Model) fieldcb.MtsItem {
	ownerName := ""
	c, err := character.NewProcessor(l, ctx).GetById()(w.CharacterId())
	if err != nil {
		l.WithError(err).Warnf("Unable to resolve owner name for want-ad character [%d]; rendering with empty seller column.", w.CharacterId())
	} else {
		ownerName = c.Name()
	}
	return mtswish.ToMtsItemWithSeller(w, ownerName)
}
