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

// walletCurrencyPrepaid is the wallet.Model.Balance convention pinned by
// wallet/model_test.go:59-61 in atlas-cashshop: currency code 3 = NX
// Prepaid. Defined locally because BUY_OTHER_PACKAGE's wire body carries no
// currency signal at all to resolve -- see handleBuyOtherPackage.
const walletCurrencyPrepaid = uint32(3)

// giftPackageFailureReasonConfigured mirrors giftFailureReasonConfigured
// (cash_shop_gift.go) for the GIFT_PACKAGE_FAILED arm: it reports whether
// this tenant's cash shop "errors" table binds an alias for the given
// BUY_OTHER_PACKAGE rejection reason. A tenant that does not bind the key
// still gets an answer -- a wedged cash shop dialog is worse than a generic
// code -- but the gap is logged for operators.
var giftPackageFailureReasonConfigured = func(l logrus.FieldLogger, ctx context.Context, reason string) bool {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), cashcb.CashShopOperationWriter)
	if !ok {
		l.Warnf("Writer options for [%s] missing; gift package failure reason [%s] not resolvable.", cashcb.CashShopOperationWriter, reason)
		return false
	}
	return atlaspacket.CodeConfigured(opts, "errors", reason)
}

// announceGiftPackageFailure emits the BUY_OTHER_PACKAGE reply arm
// (GIFT_PACKAGE_FAILED, derivation.md D3b/§5) -- NOT GiftFailedBody, which
// answers plain GIFT (mode 4). reason is a key into the tenant's "errors"
// code table, not display text.
func announceGiftPackageFailure(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, reason string) {
	return func(s session.Model, reason string) {
		if !giftPackageFailureReasonConfigured(l, ctx, reason) {
			t := tenant.MustFromContext(ctx)
			l.Warnf("Template [%s %d.%d] has no errors-table alias for gift package failure reason [%s]; sending it anyway for character [%d].", t.Region(), t.MajorVersion(), t.MinorVersion(), reason, s.CharacterId())
		}
		if err := session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopGiftPackageFailedBody(reason))(s); err != nil {
			l.WithError(err).Errorf("Unable to announce gift package failure to character [%d].", s.CharacterId())
		}
	}
}

// handleBuyPackage implements BUY_PACKAGE (mode 30/32): the buy-for-self
// half of the shared REQUEST_PACKAGE_PURCHASE command task 16 built on the
// atlas-cashshop side (RecipientCharacterId == 0 means "deliver into the
// buyer's own compartment"). The reply arrives asynchronously as a
// PACKAGE_PURCHASED status event (kafka/consumer/cashshop/consumer.go's
// handleStatusEventPackagePurchased), the same fire-and-forget shape every
// other REQUEST_* command in this handler uses -- there is no synchronous
// FAILED answer here for a producer error (only logged), matching
// handleGift's RequestGiftPurchase call.
//
// ShopOperationBuyPackage's body carries pointType bool + a legacy option
// uint32 + serialNumber (shop_operation_buy_package.go:61) -- there is no
// currency int on the wire. RequestPackagePurchase resolves the outgoing
// currency the same way the generic BUY arm's RequestPurchase does:
// resolvePurchaseCurrency(isPoints, currency) (cashshop/processor.go), so
// this call passes sp.PointType() as isPoints and the literal 0 as currency
// -- there is no raw currency to forward. option carries the user's actual
// payment-method choice (derivation.md D4a, §6: 1 = NX Credit, 2 = Maple
// Point, 4 = NX Prepaid) and is currently unconsumed server-side --
// pointType alone collapses NX Credit and NX Prepaid into the same bit, so
// passing sp.Option() through here (in place of the literal 0) would change
// the resolved currency's meaning underneath resolvePurchaseCurrency's
// non-zero passthrough without validating it first; that mapping is
// unvalidated and out of scope for this task. option is deliberately NOT
// read here for that reason -- it is not unused/spare, only not yet wired.
func handleBuyPackage(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *cashsb.ShopOperationBuyPackage) {
	return func(s session.Model, sp *cashsb.ShopOperationBuyPackage) {
		transactionId := uuid.New()
		if err := cashshop.NewProcessor(l, ctx).RequestPackagePurchase(s.CharacterId(), transactionId, sp.PointType(), 0, sp.SerialNumber(), 0, ""); err != nil {
			l.WithError(err).Errorf("Unable to request package purchase for character [%d] serial number [%d].", s.CharacterId(), sp.SerialNumber())
		}
	}
}

// handleBuyOtherPackage implements BUY_OTHER_PACKAGE (mode 31/33), the gift
// half of REQUEST_PACKAGE_PURCHASE (RecipientCharacterId != 0). Recipient
// resolution and edge validation exactly follow handleGift (cash_shop_gift.go):
// secondary credential, unknown/cross-world recipient, and no self-gifting
// are all validated here, on the wire session, BEFORE any Kafka command is
// sent -- reusing giftRejectionReason rather than a second mapper. Every
// rejection answers on GIFT_PACKAGE_FAILED (announceGiftPackageFailure),
// not GIFT_FAILED, per derivation.md D3b (§5).
//
// derivation.md D3a (§4): CCashShop::OnGiftPackage's body is spw string,
// serialNumber uint32, name string, message string -- there is no
// pointType/option field to decode at all (contrast handleBuyPackage), and
// no birthday field either, so the secondary-credential check is called
// with a literal 0 birthday; that argument is inert whenever the tenant
// uses PIC-based verification (credentialMatches ignores it), which is the
// only mode this wire body was ever observed under (GMS v95.0).
//
// The commodity's currency is NOT resolved through resolvePurchaseCurrency
// at all: D3a's closing paragraph pins CCashShop::OnGiftPackage to NX
// Prepaid only (dwOption = 4 is never encoded), so there is no
// isPoints/option signal on this wire to resolve in the first place --
// walletCurrencyPrepaid (3) is passed straight through as a hardcoded
// constant, not a value resolvePurchaseCurrency computed.
func handleBuyOtherPackage(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *cashsb.ShopOperationBuyOtherPackage) {
	return func(s session.Model, sp *cashsb.ShopOperationBuyOtherPackage) {
		if cErr := verifySecondaryCredential(l, ctx)(s, sp.SPW(), 0); cErr != nil {
			if errors.Is(cErr, ErrCredentialMismatch) {
				announceGiftPackageFailure(l, ctx, wp)(s, giftRejectionReason(cErr))
				return
			}
			l.WithError(cErr).Errorf("Unable to verify secondary credential for character [%d] requesting package gift.", s.CharacterId())
			return
		}

		recipient, err := character.NewProcessor(l, ctx).GetByName(sp.Name())
		if err != nil {
			l.WithError(err).Infof("Character [%d] attempted to gift a package to unknown recipient [%s].", s.CharacterId(), sp.Name())
			announceGiftPackageFailure(l, ctx, wp)(s, giftRejectionReason(err))
			return
		}
		if recipient.WorldId() != s.WorldId() {
			l.Infof("Character [%d] attempted to gift a package to recipient [%s] outside this world.", s.CharacterId(), sp.Name())
			announceGiftPackageFailure(l, ctx, wp)(s, giftRejectionReason(atlasmodel.ErrEmptySlice))
			return
		}

		if recipient.AccountId() == s.AccountId() {
			announceGiftPackageFailure(l, ctx, wp)(s, giftRejectionReason(errGiftOwnAccount))
			return
		}

		sender, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to resolve sender character [%d] for package gift.", s.CharacterId())
			announceGiftPackageFailure(l, ctx, wp)(s, giftRejectionReason(err))
			return
		}

		transactionId := uuid.New()
		if err := cashshop.NewProcessor(l, ctx).RequestPackagePurchase(s.CharacterId(), transactionId, false, walletCurrencyPrepaid, sp.SerialNumber(), recipient.Id(), sender.Name()); err != nil {
			l.WithError(err).Errorf("Unable to request package gift purchase for character [%d] recipient [%d].", s.CharacterId(), recipient.Id())
		}
	}
}
