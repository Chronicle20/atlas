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
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopEntryHandle = "CashShopEntryHandle"

func CashShopEntryHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		l.Debugf("Character [%d] is attempting to enter the cash shop. update_time [%d].", s.CharacterId(), updateTime)

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
		c, err := cp.GetById(cp.InventoryDecorator, cp.SkillModelDecorator, cp.PetModelDecorator)(s.CharacterId())
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

		err = session.Announce(l)(ctx)(wp)(writer.CashShopOpen)(writer.CashShopOpenBody(l)(t, a, c, bl))(s)
		if err != nil {
			return
		}

		// TODO select correct compartment
		ccp, err := compartment.NewProcessor(l, ctx).GetByAccountIdAndType(s.AccountId(), compartment.TypeExplorer)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve compartment for character [%d].", s.CharacterId())
			ccp = compartment.Model{}
		}

		err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopCashInventoryBody(l)(a, s.CharacterId(), ccp.Assets()))(s)
		if err != nil {
			return
		}

		//err = session.Announce(l)(wp)(writer.CashShopOperation)(s, writer.CashShopCashGiftsBody(l)(s.Tenant()))
		//if err != nil {
		//	return
		//}

		wl, err := wishlist.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
			return
		}
		err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopWishListBody(l)(t)(false, wl))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
		}

		w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
			w = wallet.Model{}
		}
		err = session.Announce(l)(ctx)(wp)(writer.CashShopCashQueryResult)(writer.CashShopCashQueryResultBody(t)(w.Credit(), w.Points(), w.Prepaid()))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce cash shop wallet to character [%d].", s.CharacterId())
			return

		}

		err = cashshop.NewProcessor(l, ctx).Enter(s.CharacterId(), s.Map())
		if err != nil {
			l.WithError(err).Errorf("Unable to announce [%d] has entered the cash shop.", s.CharacterId())
		}
	}
}
