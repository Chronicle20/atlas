package handler

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/inventory"
	"atlas-channel/cashshop/inventory/item"
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

		a, err := account.GetById(l)(ctx)(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate account [%d] attempting to enter cash shop.", s.AccountId())
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}
		c, err := character.GetByIdWithInventory(l)(ctx)(character.SkillModelDecorator(l)(ctx), character.PetModelDecorator(l)(ctx))(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] attempting to enter cash shop.", s.CharacterId())
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}
		bl, err := buddylist.GetById(l)(ctx)(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to locate buddylist [%d] attempting to enter cash shop.", s.CharacterId())
			_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			return
		}

		err = session.Announce(l)(ctx)(wp)(writer.CashShopOpen)(writer.CashShopOpenBody(l)(t, a, c, bl))(s)
		if err != nil {
			return
		}

		items := make([]item.Model, 0)
		items = append(items, item.NewModel(1, 1041009, 10000255, 1))
		items = append(items, item.NewModel(2, 1041071, 10000256, 1))

		inv := inventory.NewModel(items)

		err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopCashInventoryBody(l)(a, s.CharacterId(), inv))(s)
		if err != nil {
			return
		}

		//err = session.Announce(l)(wp)(writer.CashShopOperation)(s, writer.CashShopCashGiftsBody(l)(s.Tenant()))
		//if err != nil {
		//	return
		//}

		wl, err := wishlist.GetByCharacterId(l)(ctx)(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
			return
		}
		err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopWishListBody(l)(t)(false, wl))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
		}

		w, err := wallet.GetByCharacterId(l)(ctx)(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
			err = session.Announce(l)(ctx)(wp)(writer.CashShopCashQueryResult)(writer.CashShopCashQueryResultBody(t)(0, 0, 0))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce default cash shop wallet to character [%d].", s.CharacterId())
				return
			}
		} else {
			err = session.Announce(l)(ctx)(wp)(writer.CashShopCashQueryResult)(writer.CashShopCashQueryResultBody(t)(w.Credit(), w.Points(), w.Prepaid()))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash shop wallet to character [%d].", s.CharacterId())
				return
			}
		}

		err = cashshop.Enter(l)(ctx)(s.CharacterId(), s.Map())
		if err != nil {
			l.WithError(err).Errorf("Unable to announce [%d] has entered the cash shop.", s.CharacterId())
		}
	}
}
