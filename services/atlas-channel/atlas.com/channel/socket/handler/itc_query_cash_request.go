package handler

import (
	"atlas-channel/cashshop/wallet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// ItcQueryCashRequestHandleFunc handles the bodiless ITC_QUERY_CASH_REQUEST
// (CITC::TrySendQueryCashRequest) — the MTS wallet-balance query. It reads the
// two-bucket cash wallet by the session's accountId and replies with
// MTS_OPERATION2 (CITC::OnQueryCashResult): two i32 buckets, prepaid NX-cash and
// MaplePoints. This is the request/response counterpart of the cash-shop
// CashQueryResult; the MTS UI shows only the prepaid + points buckets.
func ItcQueryCashRequestHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := fieldsb.ItcQueryCashRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve MTS wallet for account [%d].", s.AccountId())
			w = wallet.Model{}
		}

		err = session.Announce(l)(ctx)(wp)(fieldcb.MtsOperation2Writer)(fieldcb.NewMtsOperation2(w.Prepaid(), w.Points()).Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce MTS wallet to character [%d].", s.CharacterId())
		}
	}
}
