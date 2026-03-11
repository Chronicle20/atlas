package handler

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/inventory/compartment"
	"atlas-channel/cashshop/wallet"
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/storage"
	"context"

	cash2 "github.com/Chronicle20/atlas-packet/cash"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CashShopEntryHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cash2.ShopEntry{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// TODO block when performing vega scrolling
		// TODO block when in event
		// TODO block when in mini dungeon
		// TODO block when already in cash shop

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

		err = session.Announce(l)(ctx)(wp)(cash2.CashShopOpenWriter)(writer.CashShopOpenBody(a, c, bl))(s)
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

		items := make([]cash2.CashInventoryItem, len(ccp.Assets()))
		for i, as := range ccp.Assets() {
			items[i] = cash2.CashInventoryItem{
				CashId:      as.Item().CashId(),
				AccountId:   a.Id(),
				CharacterId: s.CharacterId(),
				TemplateId:  as.Item().TemplateId(),
				CommodityId: as.CommodityId(),
				Quantity:    int16(as.Item().Quantity()),
				GiftFrom:    "",
				Expiration:  packetmodel.MsTime(as.Expiration()),
			}
		}
		err = session.Announce(l)(ctx)(wp)(cash2.CashShopOperationWriter)(cash2.CashShopCashInventoryBody(items, uint16(sd.Capacity), a.CharacterSlots()))(s)
		if err != nil {
			return
		}

		//err = session.Announce(l)(wp)(cash2.CashShopOperationWriter)(s, writer.CashShopCashGiftsBody(l)(s.Tenant()))
		//if err != nil {
		//	return
		//}

		wl, err := wishlist.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
			return
		}
		sns := make([]uint32, len(wl))
		for i, w := range wl {
			sns[i] = w.SerialNumber()
		}
		err = session.Announce(l)(ctx)(wp)(cash2.CashShopOperationWriter)(cash2.CashShopWishListBody(false, sns))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
		}

		w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
			w = wallet.Model{}
		}
		err = session.Announce(l)(ctx)(wp)(cash2.CashQueryResultWriter)(cash2.NewCashQueryResult(w.Credit(), w.Points(), w.Prepaid()).Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce cash shop wallet to character [%d].", s.CharacterId())
			return

		}

		err = cashshop.NewProcessor(l, ctx).Enter(s.CharacterId(), s.Field())
		if err != nil {
			l.WithError(err).Errorf("Unable to announce [%d] has entered the cash shop.", s.CharacterId())
		}
	}
}
