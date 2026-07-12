package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// ItcStatusChargeHandleFunc handles the bodiless ITC_STATUS_CHARGE
// (CITC::OnStatusCharge) — the "Charge" button in the MTS wallet strip. The
// client sends this opcode and latches the ITC UI busy (this[6]=1), then waits
// for a ChargeParamResult reply; without it, the button appears unresponsive.
//
// The correct reply is the bodiless ChargeParamResult (CITC::OnChargeParamResult,
// IDA v83 0x5a4241, dispatched by CITC::OnPacket case 346): it clears the busy
// latch (this[6]=0) and opens the client's cash-charge web page. There is no
// server-side NX-purchase transaction — NX is granted out of band — so this is
// purely the client-side charge affordance the button is meant to trigger. (An
// earlier version re-announced the wallet here, which cleared the latch but did
// nothing visible, so the button read as broken.)
func ItcStatusChargeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := fieldsb.ItcStatusCharge{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		err := session.Announce(l)(ctx)(wp)(fieldcb.MtsChargeParamResultWriter)(fieldcb.NewMtsChargeParamResult().Encode)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to announce MTS charge-param result to character [%d] on status charge.", s.CharacterId())
		}
	}
}
