package handler

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/inventory/compartment"
	"atlas-channel/cashshop/wallet"
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/character"
	"atlas-channel/minigame"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/storage"
	"context"

	"github.com/sirupsen/logrus"

	atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func CashShopEntryHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.ShopEntry{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// TODO block when performing vega scrolling
		// TODO block when in event
		// TODO block when in mini dungeon
		// TODO block when already in cash shop

		// Block entry while seated at a mini-game (Omok / Match Cards) room: a
		// player must not migrate to the cash shop mid-room. Fail open on a
		// mini-games read error so an outage there does not break cash-shop entry.
		if inGame, mgErr := minigame.NewProcessor(l, ctx).InGame(s.CharacterId()); mgErr != nil {
			l.WithError(mgErr).Warnf("Unable to determine mini-game membership for character [%d]; allowing cash shop entry.", s.CharacterId())
		} else if inGame {
			l.Debugf("Blocking cash shop entry for character [%d] currently in a mini-game room.", s.CharacterId())
			return
		}

		a, err := account.NewProcessor(l, ctx).GetById(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate account [%d] attempting to enter cash shop.", s.AccountId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}
		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator, cp.SkillModelDecorator, cp.QuestModelDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] attempting to enter cash shop.", s.CharacterId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}
		bl, err := buddylist.NewProcessor(l, ctx).GetById(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate buddylist [%d] attempting to enter cash shop.", s.CharacterId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOpenWriter)(writer.CashShopOpenBody(a, c, bl))(s)
		if err != nil {
			return
		}

		// TODO select correct compartment
		ccp, err := compartment.NewProcessor(l, ctx).GetByAccountIdAndType(s.AccountId(), compartment.TypeExplorer)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve compartment for character [%d].", s.CharacterId())
			ccp = compartment.Model{}
		}

		sd, err := storage.NewProcessor(l, ctx).GetStorageData(s.AccountId(), s.WorldId())
		if err != nil {
			l.WithError(err).Debugf("Unable to retrieve storage data for account [%d].", s.AccountId())
			sd = storage.StorageData{Capacity: storage.DefaultStorageCapacity}
		}

		items := make([]cashcb.CashInventoryItem, len(ccp.Assets()))
		for i, as := range ccp.Assets() {
			items[i] = cashcb.CashInventoryItem{
				CashId:      as.Item().CashId(),
				AccountId:   a.Id(),
				CharacterId: s.CharacterId(),
				TemplateId:  as.Item().TemplateId(),
				CommodityId: as.CommodityId(),
				Quantity:    int16(as.Item().Quantity()),
				GiftFrom:    as.GiftFrom(),
				Expiration:  packetmodel.MsTime(as.Expiration()),
			}
		}
		err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopCashInventoryBody(items, uint16(sd.Capacity), a.CharacterSlots()))(s)
		if err != nil {
			return
		}

		if loadGiftDoneConfigured(l, ctx) {
			gifts := buildGiftListEntries(ccp.Assets())
			err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopLoadGiftDoneBody(gifts))(s)
			if err != nil {
				return
			}

			// Drain the "gift list presented" flag on exactly the cashIds
			// just announced (task-240 Defect H): the client shows a modal
			// for every entry sent and sends nothing on cancel, so the
			// announce itself is the only trigger that fires exactly once
			// per presentation. Skipped entirely when the list is empty --
			// nothing to drain.
			if len(gifts) > 0 {
				cashIds := make([]int64, len(gifts))
				for i, g := range gifts {
					cashIds[i] = g.SN
				}
				if ackErr := cashshop.NewProcessor(l, ctx).AcknowledgeGifts(a.Id(), s.CharacterId(), cashIds); ackErr != nil {
					l.WithError(ackErr).Errorf("Unable to acknowledge [%d] gift(s) for character [%d].", len(cashIds), s.CharacterId())
				}
			}
		}

		wl, err := wishlist.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
			return
		}
		sns := make([]uint32, len(wl))
		for i, w := range wl {
			sns[i] = w.SerialNumber()
		}
		err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopWishListLoadBody(sns))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
		}

		w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
			w = wallet.Model{}
		}
		err = session.Announce(l)(ctx)(wp)(cashcb.CashQueryResultWriter)(cashcb.NewCashQueryResult(w.Credit(), w.Points(), w.Prepaid()).Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce cash shop wallet to character [%d].", s.CharacterId())
			return

		}

		err = cashshop.NewProcessor(l, ctx).Enter(s.CharacterId(), s.Field())
		if err != nil {
			l.WithError(err).Errorf("Unable to announce [%d] has entered the cash shop.", s.CharacterId())
		}
		_ = session.NewProcessor(l, ctx).SetCashScene(s.SessionId(), session.CashSceneCashShop)
	}
}

// buildGiftListEntries builds one cashcb.GiftListEntry per locker asset that
// carries a sender (a gift received by this character), for the
// LOAD_GIFT_SUCCESS announce. Assets with no sender (everything the
// character bought for itself) are omitted, as is any asset already marked
// GiftAcknowledged -- its gift list has already been presented in a prior
// cash shop entry (task-240 Defect H), so re-announcing it would re-fire the
// client's modal on every entry.
func buildGiftListEntries(assets []asset.Model) []cashcb.GiftListEntry {
	var gifts []cashcb.GiftListEntry
	for _, as := range assets {
		if as.GiftFrom() == "" || as.GiftAcknowledged() {
			continue
		}
		gifts = append(gifts, cashcb.GiftListEntry{
			SN:               as.Item().CashId(),
			ItemId:           int32(as.Item().TemplateId()),
			BuyCharacterName: as.GiftFrom(),
			Text:             as.GiftMessage(),
		})
	}
	return gifts
}

// loadGiftDoneConfigured reports whether this tenant's cash shop "operations"
// code table binds LOAD_GIFT_SUCCESS. template_gms_12_1.json and
// template_gms_48_1.json have a CashShopOperation writer but no
// LOAD_GIFT_SUCCESS key (docs/packets/dispatchers/cash_shop_operation.yaml's
// modes map starts at gms_v61); announcing unconditionally would fall
// through ResolveCode's 99 sentinel and push a garbage mode at those
// clients.
func loadGiftDoneConfigured(l logrus.FieldLogger, ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), cashcb.CashShopOperationWriter)
	if !ok {
		l.Warnf("Writer options for [%s] missing; LOAD_GIFT_SUCCESS not resolvable.", cashcb.CashShopOperationWriter)
		return false
	}
	return atlaspacket.CodeConfigured(opts, "operations", cashcb.CashShopOperationLoadGiftDone)
}
