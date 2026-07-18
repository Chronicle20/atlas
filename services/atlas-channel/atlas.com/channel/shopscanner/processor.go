package shopscanner

import (
	"atlas-channel/character"
	"atlas-channel/consumable"
	"atlas-channel/merchant"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

// Search executes an owl search: world-scoped merchant lookup, owner-name
// resolution, fire-and-forget count increment, mode-6 result write, and —
// only when at least one listing came back — owl consumption (design §2 Q3:
// consume 1 per search with >=1 result; empty search consumes nothing).
func (p *Processor) Search(wp writer.Producer) func(s session.Model, searchItemId uint32, descending bool, owlItemId item.Id, source slot.Position, updateTime uint32) error {
	return func(s session.Model, searchItemId uint32, descending bool, owlItemId item.Id, source slot.Position, updateTime uint32) error {
		if !_map.IsFreeMarketRoom(s.MapId()) {
			// The client cannot send this honestly (RunShopScanner hard-blocks
			// outside FM) — packet injection; drop.
			p.l.Warnf("Character [%d] attempted an owl search outside the Free Market (map [%d]).", s.CharacterId(), s.MapId())
			return nil
		}

		if searchItemId == 0 {
			// The pre-v83 owl-use frame carries no search target (the item is
			// used straight from the inventory with no CUIShopScanner input
			// dialog), so searchItemId decodes as 0. There is nothing to search
			// — drop rather than poison the hot list with item 0 or emit an
			// empty result (task-127 legacy v61/72/79). The modern cash/dedicated
			// routes always carry a real item id.
			p.l.Debugf("Character [%d] owl use with no search target (legacy frame); nothing to search.", s.CharacterId())
			return nil
		}

		mp := merchant.NewProcessor(p.l, p.ctx)

		// Count increment is result-independent (every executed search counts)
		// and must never block or fail the search.
		if err := mp.RecordItemSearch(s.Field(), s.CharacterId(), searchItemId); err != nil {
			p.l.WithError(err).Warnf("Unable to record item search for character [%d], item [%d].", s.CharacterId(), searchItemId)
		}

		listings, err := mp.SearchListings(s.WorldId(), searchItemId, descending)
		if err != nil {
			p.l.WithError(err).Errorf("Owl search failed for character [%d], item [%d]; sending empty result.", s.CharacterId(), searchItemId)
			listings = nil
		}

		names := p.resolveOwnerNames(listings)
		records := writer.ShopScannerRecords(listings, names)

		p.l.Debugf("Character [%d] owl search for item [%d]: [%d] results.", s.CharacterId(), searchItemId, len(records))
		if err := session.Announce(p.l)(p.ctx)(wp)(merchantcb.ShopScannerResultWriter)(writer.ShopScannerResultBody(searchItemId, records))(s); err != nil {
			p.l.WithError(err).Errorf("Unable to announce shop scanner result to character [%d].", s.CharacterId())
			return err
		}

		if len(listings) > 0 {
			if err := consumable.NewProcessor(p.l, p.ctx).RequestItemConsume(s.Field(), characterconst.Id(s.CharacterId()), owlItemId, source, updateTime); err != nil {
				p.l.WithError(err).Errorf("Unable to consume owl [%d] for character [%d].", owlItemId, s.CharacterId())
			}
		}

		GetRegistry().SetLastSearch(p.t, s.CharacterId(), searchItemId)
		return nil
	}
}

// resolveOwnerNames resolves distinct owner ids to names, deduplicated per
// request; a failed lookup degrades to empty string for that row.
func (p *Processor) resolveOwnerNames(listings []merchant.SearchListing) map[uint32]string {
	names := make(map[uint32]string)
	cp := character.NewProcessor(p.l, p.ctx)
	for _, sl := range listings {
		if _, ok := names[sl.OwnerId()]; ok {
			continue
		}
		c, err := cp.GetById()(sl.OwnerId())
		if err != nil {
			p.l.WithError(err).Warnf("Unable to resolve owner name for character [%d].", sl.OwnerId())
			names[sl.OwnerId()] = ""
			continue
		}
		names[sl.OwnerId()] = c.Name()
	}
	return names
}

// SendHotList answers OWL_ACTION mode OPEN with the mode-7 most-searched list.
func (p *Processor) SendHotList(wp writer.Producer) func(s session.Model) error {
	return func(s session.Model) error {
		top, err := merchant.NewProcessor(p.l, p.ctx).GetTopSearches(s.WorldId())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to fetch top searches for world [%d]; sending empty hot list.", s.WorldId())
			top = nil
		}
		itemIds := make([]uint32, 0, len(top))
		for _, ts := range top {
			itemIds = append(itemIds, ts.ItemId())
		}
		return session.Announce(p.l)(p.ctx)(wp)(merchantcb.ShopScannerResultWriter)(writer.ShopScannerHotListBody(itemIds))(s)
	}
}
