package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	atlasmodel "github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// errGiftOwnAccount is the sentinel handleGift feeds to giftRejectionReason
// when the resolved recipient's account matches the sender's own account
// (FR-GIFT-3). It never crosses a Kafka boundary or leaves this file.
var errGiftOwnAccount = errors.New("cannot gift to own account")

// giftRejectionReason maps an edge-validation rejection to its errors-table
// key. Every key is bound in template_gms_95_1.json's "errors" table:
// INCORRECT_NAME (7), CANNOT_GIFT_TO_OWN_ACCOUNT (6), INVALID_BIRTHDAY (34),
// unknown_error (69).
func giftRejectionReason(err error) string {
	switch {
	case errors.Is(err, atlasmodel.ErrEmptySlice):
		return "INCORRECT_NAME"
	case errors.Is(err, errGiftOwnAccount):
		return "CANNOT_GIFT_TO_OWN_ACCOUNT"
	case errors.Is(err, ErrCredentialMismatch):
		return "INVALID_BIRTHDAY"
	default:
		return "unknown_error"
	}
}

// giftFailureReasonConfigured reports whether this tenant's cash shop
// "errors" table binds an alias for the given GIFT rejection reason, the
// same predicate transferFailureReasonConfigured applies for
// BUY_WORLD_TRANSFER (cash_shop_operation.go). A tenant that does not bind
// the key still gets an answer -- a wedged cash shop dialog is worse than a
// generic code -- but the gap is logged for operators.
var giftFailureReasonConfigured = func(l logrus.FieldLogger, ctx context.Context, reason string) bool {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), cashcb.CashShopOperationWriter)
	if !ok {
		l.Warnf("Writer options for [%s] missing; gift failure reason [%s] not resolvable.", cashcb.CashShopOperationWriter, reason)
		return false
	}
	return atlaspacket.CodeConfigured(opts, "errors", reason)
}

// announceGiftFailure emits the real GIFT_FAILED arm. reason is a key into
// the tenant's "errors" code table, not display text.
func announceGiftFailure(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, reason string) {
	return func(s session.Model, reason string) {
		if !giftFailureReasonConfigured(l, ctx, reason) {
			t := tenant.MustFromContext(ctx)
			l.Warnf("Template [%s %d.%d] has no errors-table alias for gift failure reason [%s]; sending it anyway for character [%d].", t.Region(), t.MajorVersion(), t.MinorVersion(), reason, s.CharacterId())
		}
		if err := session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopGiftFailedBody(reason))(s); err != nil {
			l.WithError(err).Errorf("Unable to announce gift failure to character [%d].", s.CharacterId())
		}
	}
}

// handleGift implements GIFT (mode 4): FR-GIFT-1 (unknown recipient),
// FR-GIFT-2 (secondary credential), and FR-GIFT-3 (no self-gifting) are all
// validated here, on the wire session, BEFORE any Kafka command is sent --
// no command may escape on a credential-mismatch or edge-rejection path
// (task 12's ruling, reaffirmed here). Every rejection answers on the
// GIFT_FAILED arm (cashcb.CashShopGiftFailedBody), unlike BUY_NAME_CHANGE
// (handleBuyNameChange), which has no dedicated failure body and falls back
// to pink text.
//
// oneADay (sp.OneADay()) is deliberately NOT read here.
// docs/tasks/task-240-cash-shop-stub-operations/derivation.md §7 (D4b)
// resolves it as a client-set request marker (set only by
// CCSWnd_OneADay::OnButtonClicked) -- the daily state itself is server-owned
// (CCashShop::OnOneADay) and NOT determinable from the client, so there is
// no errors-table key or enforcement rule this handler could invent without
// violating the no-invented-values rule. derivation.md:9 assigns actual
// per-day-limit enforcement to Task 20; this handler's job is only to not
// silently drop the field.
func handleGift(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *cashsb.ShopOperationGift) {
	return func(s session.Model, sp *cashsb.ShopOperationGift) {
		if cErr := verifySecondaryCredential(l, ctx)(s, sp.SPW(), sp.Birthday()); cErr != nil {
			if errors.Is(cErr, ErrCredentialMismatch) {
				announceGiftFailure(l, ctx, wp)(s, giftRejectionReason(cErr))
				return
			}
			l.WithError(cErr).Errorf("Unable to verify secondary credential for character [%d] requesting gift.", s.CharacterId())
			return
		}

		recipient, err := character.NewProcessor(l, ctx).GetByName(sp.Name())
		if err != nil {
			l.WithError(err).Infof("Character [%d] attempted to gift unknown recipient [%s].", s.CharacterId(), sp.Name())
			announceGiftFailure(l, ctx, wp)(s, giftRejectionReason(err))
			return
		}
		if recipient.WorldId() != s.WorldId() {
			l.Infof("Character [%d] attempted to gift recipient [%s] outside this world.", s.CharacterId(), sp.Name())
			announceGiftFailure(l, ctx, wp)(s, giftRejectionReason(atlasmodel.ErrEmptySlice))
			return
		}

		if recipient.AccountId() == s.AccountId() {
			announceGiftFailure(l, ctx, wp)(s, giftRejectionReason(errGiftOwnAccount))
			return
		}

		sender, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to resolve sender character [%d] for gift.", s.CharacterId())
			announceGiftFailure(l, ctx, wp)(s, giftRejectionReason(err))
			return
		}

		transactionId := uuid.New()
		if err := cashshop.NewProcessor(l, ctx).RequestGiftPurchase(s.CharacterId(), transactionId, sp.SerialNumber(), recipient.Id(), sender.Name(), sp.Message()); err != nil {
			l.WithError(err).Errorf("Unable to request gift purchase for character [%d] recipient [%d].", s.CharacterId(), recipient.Id())
		}
	}
}
