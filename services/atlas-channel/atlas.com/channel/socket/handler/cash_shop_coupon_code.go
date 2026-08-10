package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	couponrules "github.com/Chronicle20/atlas/libs/atlas-constants/coupon"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// couponRedemptionRequestFunc is the seam the handler publishes through, so the
// handler's local-rejection and normalization behaviour can be tested without a
// Kafka broker (precedent: cashItemInSlotFunc in character_cash_item_use.go).
var couponRedemptionRequestFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, code string) error {
	return cashshop.NewProcessor(l, ctx).RequestCouponRedemption(characterId, code)
}

// CashShopCouponCodeHandleFunc handles the standalone serverbound COUPON_CODE
// op. This is NOT an arm of CashShopOperationHandle: the coupon submission has
// its own opcode and no leading mode byte, so it is registered by name in
// main.go like any other standalone handler.
func CashShopCouponCodeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.CouponCode{}
		p.Decode(l, ctx)(r, readerOptions)
		// p.String() reports the code's LENGTH, never its value — codes are
		// redeemable bearer tokens, and a code in a log line is a code anyone
		// with log access can spend.
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// Normalize once, here, so the value sent to atlas-cashshop and the
		// value stored in the database have the same shape. The service
		// normalizes again defensively.
		code := couponrules.Normalize(p.Code())
		if !couponrules.Plausible(code) {
			// An empty or over-long code cannot match any stored coupon, so it
			// is answered locally with no Kafka round trip. This gate is
			// load-bearing rather than an optimization: gms_v48's coupon dialog
			// has no client-side empty-code guard at all (no trim, no length
			// test — it sends straight from Confirm()==1), so on that version
			// the server is the only thing stopping an empty submission.
			err := session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(
				cashcb.CashShopUseCouponFailedBody(cashcb.CashShopOperationErrorInvalidCouponCode))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce coupon rejection to character [%d].", s.CharacterId())
			}
			return
		}

		// p.TargetCharacter() is deliberately ignored: targeted / gift coupons
		// are out of scope, the plain redeem path always sends it empty, and
		// whether a populated value carries a character name or a numeric id as
		// text is unverified. p.Type() (the jms_v185-only byte) is likewise not
		// forwarded — its semantics are unverified.
		if err := couponRedemptionRequestFunc(l, ctx, s.CharacterId(), code); err != nil {
			l.WithError(err).Errorf("Unable to request coupon redemption for character [%d].", s.CharacterId())
		}
	}
}
