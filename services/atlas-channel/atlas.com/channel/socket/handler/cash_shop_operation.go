package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/purchaserecord"
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/character"
	"atlas-channel/data/commodity"
	messageCashShop "atlas-channel/kafka/message/cashshop"
	"atlas-channel/pendingchange"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	CashShopOperationBuy                   = "BUY"                      // 3
	CashShopOperationGift                  = "GIFT"                     // 4
	CashShopOperationSetWishlist           = "SET_WISHLIST"             // 5
	CashShopOperationIncreaseInventory     = "INCREASE_INVENTORY"       // 6
	CashShopOperationIncreaseStorage       = "INCREASE_STORAGE"         // 7
	CashShopOperationIncreaseCharacterSlot = "INCREASE_CHARACTER_SLOT"  // 8
	CashShopOperationEnableEquipSlot       = "ENABLE_EQUIP_SLOT"        // 9
	CashShopOperationMoveFromCashInventory = "MOVE_FROM_CASH_INVENTORY" // 13
	CashShopOperationMoveToCashInventory   = "MOVE_TO_CASH_INVENTORY"   // 14
	CashShopOperationBuyNormal             = "BUY_NORMAL"               // 20
	CashShopOperationRebateLockerItem      = "REBATE_LOCKER_ITEM"       // 26
	CashShopOperationBuyCouple             = "BUY_COUPLE"               // 29
	CashShopOperationBuyPackage            = "BUY_PACKAGE"              // 30
	CashShopOperationBuyOtherPackage       = "BUY_OTHER_PACKAGE"        // 31
	CashShopOperationApplyWishlist         = "APPLY_WISHLIST"           // 33
	CashShopOperationBuyFriendship         = "BUY_FRIENDSHIP"           // 35
	CashShopOperationGetPurchaseRecord     = "GET_PURCHASE_RECORD"      // 40
	CashShopOperationBuyNameChange         = "BUY_NAME_CHANGE"          // 46
	CashShopOperationBuyWorldTransfer      = "BUY_WORLD_TRANSFER"       // 49
)

func CashShopOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.ShopOperation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		op := p.Op()
		var err error
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuy) {
			sp := &cashsb.ShopOperationBuy{}
			sp.Decode(l, ctx)(r, readerOptions)
			_ = cashshop.NewProcessor(l, ctx).RequestPurchase(s.CharacterId(), sp.SerialNumber(), sp.IsPoints(), sp.Currency(), sp.Zero(), uuid.Nil, "")
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationGift) {
			sp := &cashsb.ShopOperationGift{}
			sp.Decode(l, ctx)(r, readerOptions)
			handleGift(l, ctx, wp)(s, sp)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationSetWishlist) {
			sp := &cashsb.ShopOperationSetWishlist{}
			sp.Decode(l, ctx)(r, readerOptions)
			var wl []wishlist.Model
			wl, err = wishlist.NewProcessor(l, ctx).SetForCharacter(s.CharacterId(), sp.SerialNumbers())
			if err != nil {
				l.WithError(err).Errorf("Cash Shop Operation [%s] failed for character [%d].", CashShopOperationSetWishlist, s.CharacterId())
				return
			}
			sns := make([]uint32, len(wl))
			for i, w := range wl {
				sns[i] = w.SerialNumber()
			}
			err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopWishListUpdateBody(sns))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationIncreaseInventory) {
			sp := &cashsb.ShopOperationIncreaseInventory{}
			sp.Decode(l, ctx)(r, readerOptions)
			if !sp.Item() {
				err = cashshop.NewProcessor(l, ctx).RequestInventoryIncreasePurchaseByType(s.CharacterId(), sp.IsPoints(), sp.Currency(), sp.InventoryType())
				if err != nil {
					l.WithError(err).Errorf("Unable to request inventory increase purchase for character [%d].", s.CharacterId())
				}
			} else {
				err = cashshop.NewProcessor(l, ctx).RequestInventoryIncreasePurchaseByItem(s.CharacterId(), sp.IsPoints(), sp.Currency(), sp.SerialNumber())
				if err != nil {
					l.WithError(err).Errorf("Unable to request inventory increase purchase for character [%d].", s.CharacterId())
				}
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationIncreaseStorage) {
			sp := &cashsb.ShopOperationIncreaseStorage{}
			sp.Decode(l, ctx)(r, readerOptions)
			if !sp.Item() {
				err = cashshop.NewProcessor(l, ctx).RequestStorageIncreasePurchase(s.CharacterId(), sp.IsPoints(), sp.Currency())
				if err != nil {
					l.WithError(err).Errorf("Unable to request storage increase purchase for character [%d].", s.CharacterId())
				}
			} else {
				err = cashshop.NewProcessor(l, ctx).RequestStorageIncreasePurchaseByItem(s.CharacterId(), sp.IsPoints(), sp.Currency(), sp.SerialNumber())
				if err != nil {
					l.WithError(err).Errorf("Unable to request storage increase purchase for character [%d].", s.CharacterId())
				}
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationIncreaseCharacterSlot) {
			sp := &cashsb.ShopOperationIncreaseCharacterSlot{}
			sp.Decode(l, ctx)(r, readerOptions)
			err = cashshop.NewProcessor(l, ctx).RequestCharacterSlotIncreasePurchaseByItem(s.CharacterId(), sp.IsPoints(), sp.Currency(), sp.SerialNumber())
			if err != nil {
				l.WithError(err).Errorf("Unable to request character slot increase purchase for character [%d].", s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationEnableEquipSlot) {
			sp := &cashsb.ShopOperationEnableEquipSlot{}
			sp.Decode(l, ctx)(r, readerOptions)
			pt := cashshop.GetPointType(sp.PointType())
			l.Infof("Character [%d] enabling equip slot? via item [%d] using [%s].", s.CharacterId(), sp.SerialNumber(), pt)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationMoveFromCashInventory) {
			sp := &cashsb.ShopOperationMoveFromCashInventory{}
			sp.Decode(l, ctx)(r, readerOptions)
			err = cashshop.NewProcessor(l, ctx).MoveFromCashInventory(s.AccountId(), s.CharacterId(), sp.SerialNumber(), sp.InventoryType(), sp.Slot())
			if err != nil {
				l.WithError(err).Errorf("Unable to move item [%d] from cash inventory to inventory [%d] slot [%d] for character [%d].", sp.SerialNumber(), sp.InventoryType(), sp.Slot(), s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationMoveToCashInventory) {
			sp := &cashsb.ShopOperationMoveToCashInventory{}
			sp.Decode(l, ctx)(r, readerOptions)
			err = cashshop.NewProcessor(l, ctx).MoveToCashInventory(s.AccountId(), s.CharacterId(), sp.SerialNumber(), sp.InventoryType())
			if err != nil {
				l.WithError(err).Errorf("Unable to move item [%d] from inventory [%d] to cash inventory for character [%d].", sp.SerialNumber(), sp.InventoryType(), s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyNormal) {
			sp := &cashsb.ShopOperationBuyNormal{}
			sp.Decode(l, ctx)(r, readerOptions)
			// ShopOperationBuyNormal carries no isPoints/currency on the v83+
			// wire -- the whole body is a bare serialNumber (shop_operation_buy_normal.go:23-28)
			// -- so, exactly like BUY_NAME_CHANGE (handleBuyNameChange above),
			// the purchase is requested with isPoints=false, currency=0: there
			// is nothing else to charge with. The transactionId is minted here
			// per click (design §8): a Kafka redelivery replays one id while a
			// genuine second click legitimately charges twice.
			if err = cashshop.NewProcessor(l, ctx).RequestPurchase(s.CharacterId(), sp.SerialNumber(), false, 0, 0, uuid.New(), messageCashShop.ErrorOperationBuyNormal); err != nil {
				l.WithError(err).Errorf("Unable to request BUY_NORMAL purchase for character [%d] serial number [%d].", s.CharacterId(), sp.SerialNumber())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationRebateLockerItem) {
			sp := &cashsb.ShopOperationRebateLockerItem{}
			sp.Decode(l, ctx)(r, readerOptions)
			if cErr := verifySecondaryCredential(l, ctx)(s, sp.SPW(), sp.Birthday()); cErr != nil {
				if errors.Is(cErr, ErrCredentialMismatch) {
					if aErr := session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopRebateFailedBody("INVALID_BIRTHDAY"))(s); aErr != nil {
						l.WithError(aErr).Errorf("Unable to announce rebate credential failure for character [%d].", s.CharacterId())
					}
					return
				}
				l.WithError(cErr).Errorf("Unable to verify secondary credential for character [%d] requesting locker rebate.", s.CharacterId())
				return
			}
			cashId := int64(sp.Unk())
			transactionId := uuid.New()
			if err = cashshop.NewProcessor(l, ctx).RequestLockerRebate(s.AccountId(), s.CharacterId(), cashId, transactionId); err != nil {
				l.WithError(err).Errorf("Unable to request locker rebate for character [%d] cash item [%d].", s.CharacterId(), cashId)
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyCouple) {
			sp := &cashsb.ShopOperationBuyCouple{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Infof("Character [%d] purchasing [%d] for [%s] with message [%s]. Option [%d], birthday [%d]", s.CharacterId(), sp.SerialNumber(), sp.Name(), sp.Message(), sp.Option(), sp.Birthday())
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyPackage) {
			sp := &cashsb.ShopOperationBuyPackage{}
			sp.Decode(l, ctx)(r, readerOptions)
			pt := cashshop.GetPointType(sp.PointType())
			l.Infof("Character [%d] purchasing [%d] with [%s]. Option [%d]", s.CharacterId(), sp.SerialNumber(), pt, sp.Option())
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationApplyWishlist) {
			// APPLY_WISHLIST (mode 33/35) carries no bytes after the mode byte
			// (derivation.md D2a: RESOLVED, empty body). The reply arm is
			// inferred (derivation.md D2b: RESOLVED but flagged INFERENTIAL,
			// reached via request-in-flight-latch analysis rather than a
			// client correlation table) to be UPDATE_WISHLIST (mode 98), the
			// same arm SET_WISHLIST already answers with.
			// The failure arm must pair with the latch-clearing UPDATE_WISHLIST
			// success arm above (derivation.md D2b evidence 1): SET_WISH_FAILED
			// (mode 99, CCashShop::OnCashItemResSetWishFailed) clears the
			// client's request-in-flight latch; LOAD_WISH_FAILED does not, and
			// answering with it would leave the client wedged. This corrects
			// the brief's Step 5 instruction, which named LOAD_WISH_FAILED in
			// conflict with the derivation it itself cites.
			wl, err := wishlist.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve wishlist for character [%d].", s.CharacterId())
				err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopSetWishFailedBody("unknown_error"))(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce wishlist load failure for character [%d].", s.CharacterId())
				}
				return
			}
			sns := make([]uint32, len(wl))
			for i, w := range wl {
				sns[i] = w.SerialNumber()
			}
			err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopWishListUpdateBody(sns))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce wishlist for character [%d].", s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyFriendship) {
			sp := &cashsb.ShopOperationBuyFriendship{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Infof("Character [%d] purchasing [%d] for [%s] with message [%s]. Option [%d], birthday [%d]", s.CharacterId(), sp.SerialNumber(), sp.Name(), sp.Message(), sp.Option(), sp.Birthday())
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationGetPurchaseRecord) {
			sp := &cashsb.ShopOperationGetPurchaseRecord{}
			sp.Decode(l, ctx)(r, readerOptions)
			m, err := purchaserecord.NewProcessor(l, ctx).GetForAccount(s.AccountId(), sp.SerialNumber())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve purchase record for account [%d] serial [%d].", s.AccountId(), sp.SerialNumber())
				err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopPurchaseRecordFailedBody("unknown_error"))(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce purchase record failure for character [%d].", s.CharacterId())
				}
				return
			}
			err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopPurchaseRecordDoneBody(int32(sp.SerialNumber()), purchaseRecordFlag(m.Count())))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce purchase record for character [%d].", s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyNameChange) {
			sp := &cashsb.ShopOperationBuyNameChange{}
			sp.Decode(l, ctx)(r, readerOptions)
			handleBuyNameChange(l, ctx, wp)(s, sp)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyWorldTransfer) {
			sp := &cashsb.ShopOperationBuyWorldTransfer{}
			sp.Decode(l, ctx)(r, readerOptions)
			handleBuyWorldTransfer(l, ctx, wp)(s, sp)
			return
		}
		l.Warnf("Unhandled Cash Shop Operation [%d] issued by character [%d].", op, s.CharacterId())
	}
}

// handleBuyNameChange implements BUY_NAME_CHANGE (mode 46): it turns the
// client's purchase-with-name request into a pending-change record in
// atlas-character — the same record the item-use/cancel arm (Task 24) can
// later cancel — and then charges the coupon through the same
// REQUEST_PURCHASE pipeline every other BUY_* arm uses.
// ShopOperationBuyNameChange carries no isPoints/currency fields on the wire
// (unlike ShopOperationBuy, ShopOperationIncreaseInventory,
// ShopOperationIncreaseStorage, ShopOperationIncreaseCharacterSlot, which all
// decode isPoints+currency), so the purchase is requested with isPoints=false,
// currency=0 — there is nothing else on the wire to charge with. The pending
// record is inserted BEFORE the purchase is requested (insert-first,
// task-227 task 38): the purchase outcome event can otherwise reach the
// consumer before this handler has finished, and the consumer resolves it
// back to the pending record via the transactionId minted here from the
// record's own Id. If that Id fails to parse as a UUID (unreachable in
// practice -- RestModel.Id is always a uuid.UUID's own String() -- but not
// handled if it somehow happened), the purchase is never requested: the
// just-created pending record is cancelled and the rejection route fires
// instead, so a parse failure can never charge the player for a purchase no
// consumer can resolve. RequestPurchase is fully async; a producer error
// does not prove non-delivery, so a failed emit is logged (for an operator
// to find the possibly-orphaned pending record) but the record is left
// alone rather than cancelled -- cancelling here risks a worse
// double-fault if the message actually got through. Every outcome (success
// or failure) that does reach the consumer arrives as a status event; this
// handler no longer answers the client itself for that path (task 39 wires
// that consumer-side answer up).
//
// The client has no NAME_CHANGE_FAILED arm (every other BUY_* has a
// *_FAILED sibling constant in shop_operation_body.go; name-change does
// not), so every rejection here goes out as pink text (FR-5.1), the same
// route the expiration-extender arm in character_cash_item_use.go uses.
func handleBuyNameChange(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *cashsb.ShopOperationBuyNameChange) {
	return func(s session.Model, sp *cashsb.ShopOperationBuyNameChange) {
		characterId := s.CharacterId()

		c, err := character.NewProcessor(l, ctx).GetById()(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] to validate name change request.", characterId)
			announceCashShopRejection(l, ctx, wp)(s, "Unable to process your name change request.")
			return
		}
		// Never trust the client's OldName() — refuse if it disagrees with
		// the session character's actual current name.
		if c.Name() != sp.OldName() {
			l.Warnf("Character [%d] requested name change claiming old name [%s] but session character is [%s]; refusing.", characterId, sp.OldName(), c.Name())
			announceCashShopRejection(l, ctx, wp)(s, "Unable to process your name change request.")
			return
		}

		// The commodity is resolved purely to reject an unusable serial number
		// BEFORE the insert-first pending record exists — an unresolvable
		// serial would otherwise strand a PENDING record when RequestPurchase
		// later fails. Its template id is deliberately not carried onto the
		// request: see pendingchange.RequestNameChange.
		if _, err := commodity.NewProcessor(l, ctx).GetById(sp.SerialNumber()); err != nil {
			l.WithError(err).Errorf("Unable to resolve commodity [%d] for character [%d] name change request.", sp.SerialNumber(), characterId)
			announceCashShopRejection(l, ctx, wp)(s, "Unable to process your name change request.")
			return
		}

		rm, err := pendingchange.NewProcessor(l, ctx).RequestNameChange(characterId, sp.NewName())
		if err != nil {
			l.WithError(err).Warnf("Name change request rejected for character [%d].", characterId)
			announceCashShopRejection(l, ctx, wp)(s, nameChangeRejectionMessage(err))
			return
		}

		transactionId, err := uuid.Parse(rm.Id)
		if err != nil {
			l.WithError(err).Errorf("Pending change record [%s] for character [%d] is not a valid UUID; aborting purchase and cancelling the record.", rm.Id, characterId)
			if _, cancelErr := pendingchange.NewProcessor(l, ctx).CancelPendingChange(characterId, pendingchange.TypeNameChange); cancelErr != nil {
				l.WithError(cancelErr).Errorf("Unable to cancel pending name change record [%s] for character [%d] after transaction id parse failure.", rm.Id, characterId)
			}
			announceCashShopRejection(l, ctx, wp)(s, "Unable to process your name change request.")
			return
		}
		if err := cashshop.NewProcessor(l, ctx).RequestPurchase(characterId, sp.SerialNumber(), false, 0, 0, transactionId, ""); err != nil {
			l.WithError(err).Errorf("Unable to request purchase for character [%d] serial number [%d] transaction [%s]; pending name change record [%s] may be orphaned.", characterId, sp.SerialNumber(), transactionId, rm.Id)
		}
	}
}

// handleBuyWorldTransfer implements BUY_WORLD_TRANSFER (mode 49). Same
// division of responsibility as handleBuyNameChange: ShopOperationBuyWorldTransfer
// carries no isPoints/currency either, so the purchase is requested with
// isPoints=false, currency=0. The pending record is inserted BEFORE the
// purchase is requested (insert-first, task-227 task 38) — see
// handleBuyNameChange's doc for why the ordering matters, how the
// transactionId correlates back to the record, and why a transactionId
// parse failure cancels the record and aborts the purchase while a
// discarded RequestPurchase emit error is only logged, never cancelled.
// Unlike name-change, a
// real client failure arm exists here (CashShopTransferWorldFailedBody /
// TRANSFER_WORLD_FAILED, shop_operation_body.go:454), so pending-change
// rejections use it instead of pink text. On the rejection-free path this
// handler also emits FR-4.7's storage-stranding courtesy warning
// (warnIfStrandingStorage, below) — moved here from the CHECK handler to
// avoid a client-side modal collision, see that function's own doc comment.
func handleBuyWorldTransfer(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *cashsb.ShopOperationBuyWorldTransfer) {
	return func(s session.Model, sp *cashsb.ShopOperationBuyWorldTransfer) {
		characterId := s.CharacterId()

		// Resolved only to reject an unusable serial number before the
		// insert-first pending record exists; the template id is deliberately
		// not carried onto the request. See handleBuyNameChange.
		if _, err := commodity.NewProcessor(l, ctx).GetById(sp.SerialNumber()); err != nil {
			l.WithError(err).Errorf("Unable to resolve commodity [%d] for character [%d] world transfer request.", sp.SerialNumber(), characterId)
			announceTransferWorldFailure(l, ctx, wp)(s, "unknown_error")
			return
		}

		destinationWorldId := world.Id(sp.TargetWorld())
		rm, err := pendingchange.NewProcessor(l, ctx).RequestWorldTransfer(characterId, destinationWorldId)
		if err != nil {
			l.WithError(err).Warnf("World transfer request rejected for character [%d].", characterId)
			announceTransferWorldFailure(l, ctx, wp)(s, worldTransferRejectionReason(err))
			return
		}

		transactionId, err := uuid.Parse(rm.Id)
		if err != nil {
			l.WithError(err).Errorf("Pending change record [%s] for character [%d] is not a valid UUID; aborting purchase and cancelling the record.", rm.Id, characterId)
			if _, cancelErr := pendingchange.NewProcessor(l, ctx).CancelPendingChange(characterId, pendingchange.TypeWorldTransfer); cancelErr != nil {
				l.WithError(cancelErr).Errorf("Unable to cancel pending world transfer record [%s] for character [%d] after transaction id parse failure.", rm.Id, characterId)
			}
			announceTransferWorldFailure(l, ctx, wp)(s, "unknown_error")
			return
		}
		if err := cashshop.NewProcessor(l, ctx).RequestPurchase(characterId, sp.SerialNumber(), false, 0, 0, transactionId, ""); err != nil {
			l.WithError(err).Errorf("Unable to request purchase for character [%d] serial number [%d] transaction [%s]; pending world transfer record [%s] may be orphaned.", characterId, sp.SerialNumber(), transactionId, rm.Id)
		}

		// FR-4.7's storage-stranding courtesy warning fires HERE, not from the
		// CHECK handler it originally shipped in. It used to be emitted right
		// after CHECK's ALLOWED result, but ALLOWED opens the client's
		// license-notice dialog as a modal (CUITransferWorldLicenseNotice ::
		// DoModal), and the warning's own POP_UP world message opens a SECOND
		// modal (CUtilDlg::Notice) inside that one's nested loop — stealing
		// the input grab and leaving the license notice dead behind it. See
		// docs/tasks/task-227-cash-name-change-world-transfer/bug-world-transfer-eligibility-reasons.md
		// (Symptom 1) for the full IDA trace and Ruling 1. By the time this
		// handler runs, the player has already dismissed that dialog and
		// picked a destination, so there is no modal left for the warning to
		// collide with — and this is also the first point the warning is
		// actually actionable.
		warnIfStrandingStorage(l, ctx, wp, s, characterId)
	}
}

// warnIfStrandingStorage implements task-227 Task 26's FR-4.7 courtesy
// notice: storage is keyed (tenant, world, account) and shared by every
// character the account owns in that world (FR-4.6), so it is only stranded
// when the transferring character is the account's LAST character in the
// SOURCE world — s.WorldId(), i.e. the world being left, not the destination
// world this BUY carries. This is advisory, not a gate, so it FAILS OPEN: a
// lookup error is logged and swallowed, never surfaced to the player and
// never allowed to affect the (already-requested) purchase.
func warnIfStrandingStorage(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, characterId uint32) {
	chars, err := checkPossibleAccountCharactersInWorldFunc(l, ctx, s.AccountId(), s.WorldId())
	if err != nil {
		l.WithError(err).Errorf("Unable to determine whether world transfer strands storage for account [%d] world [%d]; skipping the courtesy warning.", s.AccountId(), s.WorldId())
		return
	}

	isLast := len(chars) == 1 && chars[0].Id() == characterId
	if !isLast {
		return
	}

	msg := "Your Cash Shop storage in this world is tied to your account, not this character. Because this is your only remaining character here, it will become inaccessible once the transfer completes."
	// POP_UP, not PINK_TEXT. The recipient is by definition sitting in the
	// Cash Shop, where there is no status bar: CWvsContext::OnBroadcastMsg
	// @0xa22785 (GMS v83) routes PINK_TEXT (arm 5) through CHATLOG_ADD
	// @0x4906b5, whose whole body is guarded by
	// `if (TSingleton<CUIStatusBar>::ms_pInstance)` — so the warning is a
	// silent no-op there. Arm 1 (POP_UP) instead falls straight through to
	// CUtilDlg::Notice with no status-bar guard at all, which is the only
	// delivery this op's audience actually renders. Same reason
	// announceCashShopRejection (above) uses the pop-up.
	if err := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePopUpBody(msg))(s); err != nil {
		l.WithError(err).Errorf("Unable to write storage-stranding warning for character [%d].", s.CharacterId())
	}
}

// purchaseRecordFlag maps a purchase count onto the wire's single
// purchased byte (PurchaseRecordDone.purchased, compared !=0 -> bool by the
// client): any count > 0 is "purchased", never a literal count.
func purchaseRecordFlag(count uint32) byte {
	if count > 0 {
		return 1
	}
	return 0
}

// announceCashShopRejection is the FR-5.1 pink-text fallback for arms with
// no dedicated failure body — the same route the expiration-extender arm
// (character_cash_item_use.go) uses.
func announceCashShopRejection(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, message string) {
	return func(s session.Model, message string) {
		if err := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePopUpBody(message))(s); err != nil {
			l.WithError(err).Errorf("Unable to announce cash shop rejection to character [%d].", s.CharacterId())
		}
	}
}

// transferFailureReasonConfigured reports whether this tenant's cash shop
// "errors" table binds an alias for the given world-transfer rejection
// reason (see docs/tasks/task-227-cash-name-change-world-transfer/
// bug-world-transfer-eligibility-reasons.md, "Rulings" ruling 2 and its
// alias table). Most templates carry an alias for every reason; two do not:
// template_gms_12_1.json has no "errors" table at all, and
// template_jms_185_1.json's table carries none of the CANNOT_TRANSFER_*
// codes the aliases point at. This predicate turns that gap into an
// explicit, logged fallback to the client's generic notice arm instead of
// letting it fall through silently to ResolveCode's 99 sentinel.
// Package-level var so tests can drive both the bound and unbound case
// without a live writer registry (mirrors tradeEnterErrorConfigured in
// kafka/consumer/trade/consumer.go).
var transferFailureReasonConfigured = func(l logrus.FieldLogger, ctx context.Context, reason string) bool {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), cashcb.CashShopOperationWriter)
	if !ok {
		l.Warnf("Writer options for [%s] missing; world transfer failure reason [%s] not resolvable.", cashcb.CashShopOperationWriter, reason)
		return false
	}
	return atlaspacket.CodeConfigured(opts, "errors", reason)
}

// announceTransferWorldFailure emits the real TRANSFER_WORLD_FAILED arm.
// reason is a key into the tenant's "errors" code table (same convention as
// every other *_FAILED body in this family, e.g. CashShopBuyFailedBody),
// not display text.
func announceTransferWorldFailure(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, reason string) {
	return func(s session.Model, reason string) {
		if !transferFailureReasonConfigured(l, ctx, reason) {
			t := tenant.MustFromContext(ctx)
			l.Warnf("Template [%s %d.%d] has no errors-table alias for world transfer failure reason [%s]; falling back to the client's generic notice for character [%d].", t.Region(), t.MajorVersion(), t.MinorVersion(), reason, s.CharacterId())
		}
		if err := session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopTransferWorldFailedBody(reason))(s); err != nil {
			l.WithError(err).Errorf("Unable to announce world transfer failure to character [%d].", s.CharacterId())
		}
	}
}

// nameChangeRejectionMessage maps a pendingchange rejection to player-facing
// pink text. There is no NAME_CHANGE_FAILED arm to carry a coded reason (see
// handleBuyNameChange doc), so the text itself is the whole payload — never
// leak the raw machine reason key (e.g. "name_reserved") verbatim.
func nameChangeRejectionMessage(err error) string {
	var re *pendingchange.RejectedError
	if !errors.As(err, &re) {
		return "Unable to process your name change request."
	}
	switch re.Reason {
	case "already_pending":
		return "You already have a pending name change request."
	case "name_reserved":
		return "That name is already in use."
	default:
		return "Unable to process your name change request."
	}
}

// worldTransferRejectionReason maps a pendingchange rejection to an "errors"
// table key for CashShopTransferWorldFailedBody. Falls back to a generic key
// when the processor did not decode a specific reason (the empty-detail
// already-terminal case) or the failure was infrastructural.
func worldTransferRejectionReason(err error) string {
	var re *pendingchange.RejectedError
	if errors.As(err, &re) && re.Reason != "" {
		return re.Reason
	}
	return "unknown_error"
}

func isCashShopOperation(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key string) bool {
	return func(options map[string]interface{}, op byte, key string) bool {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}
		return byte(res) == op
	}
}
