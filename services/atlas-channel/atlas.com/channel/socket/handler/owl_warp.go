package handler

import (
	"atlas-channel/character"
	"atlas-channel/merchant"
	"atlas-channel/portal"
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	merchantsb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// OwlWarpHandleFunc handles CUIShopScanResult::OnButtonClicked: re-validates
// the clicked result against current shop state (design §4.2 ladder), then
// warps same-channel and stages the pending auto-enter. Every failure rung
// answers with the faithful SHOP_LINK code; success sends no packet (the
// client tears the scanner windows down on field change).
func OwlWarpHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := merchantsb.OwlWarp{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		announceLink := func(code merchantpkt.ShopLinkResultCode) {
			_ = session.Announce(l)(ctx)(wp)(merchantcb.ShopLinkResultWriter)(writer.ShopLinkResultBody(code))(s)
		}

		check := shopscanner.WarpCheck{
			OwnerId:          p.OwnerId(),
			CharacterId:      s.CharacterId(),
			CurrentMapFM:     _map.IsFreeMarketRoom(s.MapId()),
			SessionWorldId:   s.WorldId(),
			SessionChannelId: s.ChannelId(),
			EchoedMapId:      p.MapId(),
		}

		reg := shopscanner.GetRegistry()
		last, hasSearch := reg.GetLastSearch(t, s.CharacterId())
		check.HasSearch = hasSearch

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d] for owl warp.", s.CharacterId())
			announceLink(merchantpkt.ShopLinkResultCodeClosed)
			return
		}
		check.CharacterHp = c.Hp()

		mp := merchant.NewProcessor(l, ctx)
		var shopId uuid.UUID
		shops, err := mp.GetByCharacterId(p.OwnerId())
		if err == nil && len(shops) > 0 {
			shop := shops[0]
			check.ShopFound = true
			check.ShopWorldId = shop.WorldId()
			check.ShopChannelId = shop.ChannelId()
			check.ShopMapId = shop.MapId()
			check.ShopState = shop.State()
			shopId = shop.Id()
		}

		// Listing-still-present check: re-query the world-scoped search for the
		// remembered item and look for this shop with bundles remaining.
		if check.ShopFound && hasSearch {
			listings, err := mp.SearchListings(s.WorldId(), last.ItemId, false)
			if err != nil {
				l.WithError(err).Warnf("Unable to re-validate listing for owl warp of character [%d].", s.CharacterId())
			} else {
				for _, sl := range listings {
					if sl.ShopId() == shopId && sl.BundlesRemaining() > 0 {
						check.ListingPresent = true
						break
					}
				}
			}
		}

		if code, ok := shopscanner.EvaluateWarp(check); !ok {
			l.Infof("Owl warp rejected for character [%d] to owner [%d]: code [%s].", s.CharacterId(), p.OwnerId(), code)
			announceLink(code)
			return
		}

		reg.SetPending(t, s.CharacterId(), shopscanner.PendingEntry{
			ShopId:  shopId,
			OwnerId: p.OwnerId(),
			MapId:   _map.Id(p.MapId()),
		})
		l.Debugf("Character [%d] owl-warping to shop of owner [%d] in map [%d].", s.CharacterId(), p.OwnerId(), p.MapId())
		if err := portal.NewProcessor(l, ctx).Warp(s.Field(), s.CharacterId(), _map.Id(p.MapId())); err != nil {
			l.WithError(err).Errorf("Unable to warp character [%d] for owl warp.", s.CharacterId())
			reg.RemovePending(t, s.CharacterId())
			announceLink(merchantpkt.ShopLinkResultCodeClosed)
		}
	}
}
