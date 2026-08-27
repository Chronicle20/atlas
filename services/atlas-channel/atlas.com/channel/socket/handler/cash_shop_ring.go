package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"

	messagecashshop "atlas-channel/kafka/message/cashshop"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	atlasmodel "github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ringTypeForOperation is the pure mapper handleBuyCouple/handleBuyFriendship
// share to select RequestRingPurchaseCommandBody.RingType from the arm the
// wire body was decoded for -- mirrors ring.TypeCouple/ring.TypeFriendship's
// string values on the atlas-cashshop side (task 19), duplicated in this
// service as messagecashshop.RingTypeCouple/RingTypeFriendship.
func ringTypeForOperation(operation string) string {
	switch operation {
	case CashShopOperationBuyCouple:
		return messagecashshop.RingTypeCouple
	case CashShopOperationBuyFriendship:
		return messagecashshop.RingTypeFriendship
	default:
		return ""
	}
}

// ringFailureBodyForType picks CashShopCoupleFailedBody or
// CashShopFriendshipFailedBody by ringType -- the two ring arms answer on
// distinct *_FAILED mode bytes (mirroring failureBodyForOperation's routing
// for the asynchronous ERROR path in kafka/consumer/cashshop/consumer.go).
func ringFailureBodyForType(ringType string, reason string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	if ringType == messagecashshop.RingTypeFriendship {
		return cashcb.CashShopFriendshipFailedBody(reason)
	}
	return cashcb.CashShopCoupleFailedBody(reason)
}

// ringFailureReasonConfigured mirrors giftFailureReasonConfigured
// (cash_shop_gift.go) for the ring COUPLE_FAILED/FRIENDSHIP_FAILED arms: it
// reports whether this tenant's cash shop "errors" table binds an alias for
// the given rejection reason. A tenant that does not bind the key still
// gets an answer -- a wedged cash shop dialog is worse than a generic code
// -- but the gap is logged for operators.
var ringFailureReasonConfigured = func(l logrus.FieldLogger, ctx context.Context, reason string) bool {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), cashcb.CashShopOperationWriter)
	if !ok {
		l.Warnf("Writer options for [%s] missing; ring failure reason [%s] not resolvable.", cashcb.CashShopOperationWriter, reason)
		return false
	}
	return atlaspacket.CodeConfigured(opts, "errors", reason)
}

// announceRingFailure emits the real COUPLE_FAILED / FRIENDSHIP_FAILED arm
// (by ringType). reason is a key into the tenant's "errors" code table, not
// display text.
func announceRingFailure(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, ringType string, reason string) {
	return func(s session.Model, ringType string, reason string) {
		if !ringFailureReasonConfigured(l, ctx, reason) {
			t := tenant.MustFromContext(ctx)
			l.Warnf("Template [%s %d.%d] has no errors-table alias for ring failure reason [%s]; sending it anyway for character [%d].", t.Region(), t.MajorVersion(), t.MinorVersion(), reason, s.CharacterId())
		}
		if err := session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(ringFailureBodyForType(ringType, reason))(s); err != nil {
			l.WithError(err).Errorf("Unable to announce ring failure to character [%d].", s.CharacterId())
		}
	}
}

// handleRingPurchase implements BUY_COUPLE (mode 29/31) and BUY_FRIENDSHIP
// (mode 35/37): FR-RING-5 (secondary credential), FR-RING-2 (unknown
// partner name), and the no-self-partnering rule are all validated here, on
// the wire session, BEFORE any Kafka command is sent -- no command may
// escape on a credential-mismatch or edge-rejection path (mirroring
// handleGift's ruling, cash_shop_gift.go). Every rejection answers on the
// arm's own *_FAILED mode byte (announceRingFailure), keyed by ringType.
//
// OQ-R1 (context.md:188-208): the typed distinct-halves rejection branch
// (COUPLE_FAILED/FRIENDSHIP_FAILED for a pair whose halves disagree) is
// deliberately UNIMPLEMENTED here -- no data source on this wire or in the
// resolved partner/sender records lets this handler detect that condition.
//
// option (ShopOperationBuyCouple.Option/ShopOperationBuyFriendship.Option) IS
// read here, unlike handleBuyPackage's D4a treatment (derivation.md §6,
// cash_shop_package.go): on GMS >= 83 these two arms carry the user's
// confirmation-dialog payment-method choice (1 = NX Credit, 2 = Maple Point,
// 4 = NX Prepaid) only in option -- isPoints/currency stay false/0 on that
// wire shape, so without option every ring purchase silently debited the
// prepaid bucket regardless of what the player selected
// (bug-ring-purchase-wrong-currency.md). option is threaded straight through
// to cashshop.Processor.RequestRingPurchase, which resolves the final wallet
// currency via resolveRingPurchaseCurrency.
func handleRingPurchase(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, ringType string, isPoints bool, currency uint32, option uint32, spw string, birthday uint32, serialNumber uint32, name string, message string) {
	return func(s session.Model, ringType string, isPoints bool, currency uint32, option uint32, spw string, birthday uint32, serialNumber uint32, name string, message string) {
		if cErr := verifySecondaryCredential(l, ctx)(s, spw, birthday); cErr != nil {
			if errors.Is(cErr, ErrCredentialMismatch) {
				announceRingFailure(l, ctx, wp)(s, ringType, giftRejectionReason(cErr))
				return
			}
			l.WithError(cErr).Errorf("Unable to verify secondary credential for character [%d] requesting ring purchase.", s.CharacterId())
			return
		}

		partner, err := character.NewProcessor(l, ctx).GetByName(name)
		if err != nil {
			l.WithError(err).Infof("Character [%d] attempted to purchase a ring for unknown partner [%s].", s.CharacterId(), name)
			// Forced to atlasmodel.ErrEmptySlice -- see the identical
			// comment on handleGift's own recipient lookup
			// (cash_shop_gift.go), the pattern this copies (task 24a item
			// 3, flagged non-blocking in review-task-20.md).
			announceRingFailure(l, ctx, wp)(s, ringType, giftRejectionReason(atlasmodel.ErrEmptySlice))
			return
		}
		if partner.WorldId() != s.WorldId() {
			l.Infof("Character [%d] attempted to purchase a ring for partner [%s] outside this world.", s.CharacterId(), name)
			announceRingFailure(l, ctx, wp)(s, ringType, giftRejectionReason(atlasmodel.ErrEmptySlice))
			return
		}

		if partner.AccountId() == s.AccountId() {
			announceRingFailure(l, ctx, wp)(s, ringType, giftRejectionReason(errGiftOwnAccount))
			return
		}

		sender, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to resolve sender character [%d] for ring purchase.", s.CharacterId())
			announceRingFailure(l, ctx, wp)(s, ringType, giftRejectionReason(err))
			return
		}

		transactionId := uuid.New()
		if err := cashshop.NewProcessor(l, ctx).RequestRingPurchase(s.CharacterId(), transactionId, isPoints, currency, option, serialNumber, partner.Id(), sender.Name(), message, ringType); err != nil {
			l.WithError(err).Errorf("Unable to request ring purchase for character [%d] partner [%d].", s.CharacterId(), partner.Id())
		}
	}
}

// handleBuyCouple unpacks ShopOperationBuyCouple and delegates to
// handleRingPurchase with ringType "COUPLE".
func handleBuyCouple(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *cashsb.ShopOperationBuyCouple) {
	return func(s session.Model, sp *cashsb.ShopOperationBuyCouple) {
		handleRingPurchase(l, ctx, wp)(s, ringTypeForOperation(CashShopOperationBuyCouple), sp.IsPoints(), sp.Currency(), sp.Option(), sp.SPW(), sp.Birthday(), sp.SerialNumber(), sp.Name(), sp.Message())
	}
}

// handleBuyFriendship unpacks ShopOperationBuyFriendship and delegates to
// handleRingPurchase with ringType "FRIENDSHIP". ShopOperationBuyFriendship's
// v48-only flag byte (shop_operation_buy_friendship.go:26,83-90, a
// client-hard-coded constant absent on v83+) needs no handling here -- it is
// decoded but never read by this handler, exactly like option above.
func handleBuyFriendship(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *cashsb.ShopOperationBuyFriendship) {
	return func(s session.Model, sp *cashsb.ShopOperationBuyFriendship) {
		handleRingPurchase(l, ctx, wp)(s, ringTypeForOperation(CashShopOperationBuyFriendship), sp.IsPoints(), sp.Currency(), sp.Option(), sp.SPW(), sp.Birthday(), sp.SerialNumber(), sp.Name(), sp.Message())
	}
}
