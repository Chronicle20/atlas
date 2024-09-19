package handler

import (
	"atlas-channel/account"
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
	cashShopOpenFunc := session.Announce(l)(ctx)(wp)(writer.CashShopOpen)
	//cashShopOperationFunc := session.Announce(l)(wp)(writer.CashShopOperation)
	cashShopCashQueryResultFunc := session.Announce(l)(ctx)(wp)(writer.CashShopCashQueryResult)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		l.Debugf("Character [%d] is attempting to enter the cash shop. update_time [%d].", s.CharacterId(), updateTime)

		// TODO block when performing vega scrolling
		// TODO block when in event
		// TODO block when in mini dungeon
		// TODO block when already in cash shop

		a, err := account.GetById(l)(ctx)(s.AccountId())
		c, err := character.GetByIdWithInventory(l)(ctx)(s.CharacterId())

		err = cashShopOpenFunc(s, writer.CashShopOpenBody(l)(t, a, c))
		if err != nil {
			return
		}

		//err = cashShopOperationFunc(s, writer.CashShopCashInventoryBody(l)(s.Tenant()))
		//if err != nil {
		//	return
		//}
		//
		//err = cashShopOperationFunc(s, writer.CashShopCashGiftsBody(l)(s.Tenant()))
		//if err != nil {
		//	return
		//}
		//
		//err = cashShopOperationFunc(s, writer.CashShopWishListBody(l)(s.Tenant()))
		//if err != nil {
		//	return
		//}

		err = cashShopCashQueryResultFunc(s, writer.CashShopCashQueryResultBody(l)(t))
		if err != nil {
			return
		}
	}
}
