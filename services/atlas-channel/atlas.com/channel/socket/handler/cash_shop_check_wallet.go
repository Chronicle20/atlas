package handler

import (
	"atlas-channel/cashshop/wallet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	cash2 "github.com/Chronicle20/atlas-packet/cash"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CashShopCheckWalletHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cash2.CheckWallet{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
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
	}
}
