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

// ItcStatusChargeHandleFunc handles the bodiless ITC_STATUS_CHARGE
// (CITC::OnStatusCharge) — the open-NX-recharge hook the MTS UI fires when the
// player triggers the "charge cash" affordance. atlas-channel has no distinct
// server-side recharge transaction (NX is purchased out of band), so the
// realistic acknowledgment is to re-read the two-bucket wallet and re-announce
// the current balance via MTS_OPERATION2 (the same response the wallet query
// elicits), refreshing the displayed balance. This is wired to the reachable
// wallet flow rather than a silent no-op.
func ItcStatusChargeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := fieldsb.ItcStatusCharge{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve MTS wallet for account [%d] on status charge.", s.AccountId())
			w = wallet.Model{}
		}

		err = session.Announce(l)(ctx)(wp)(fieldcb.MtsOperation2Writer)(fieldcb.NewMtsOperation2(w.Prepaid(), w.Points()).Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce MTS wallet to character [%d] on status charge.", s.CharacterId())
		}
	}
}
