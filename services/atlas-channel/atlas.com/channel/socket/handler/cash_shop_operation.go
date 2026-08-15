package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/character"
	"atlas-channel/data/commodity"
	"atlas-channel/pendingchange"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
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
			_ = cashshop.NewProcessor(l, ctx).RequestPurchase(s.CharacterId(), sp.SerialNumber(), sp.IsPoints(), sp.Currency(), sp.Zero(), uuid.Nil)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationGift) {
			sp := &cashsb.ShopOperationGift{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Infof("Character [%d] gifting [%d] to [%s] with message [%s]. birthday [%d]", s.CharacterId(), sp.SerialNumber(), sp.Name(), sp.Message(), sp.Birthday())
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
			l.Infof("Character [%d] purchasing [%d].", s.CharacterId(), sp.SerialNumber())
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationRebateLockerItem) {
			sp := &cashsb.ShopOperationRebateLockerItem{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Infof("Character [%d] using rebate [%d]. birthday [%d]", s.CharacterId(), sp.Unk(), sp.Birthday())
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
			l.Infof("Character [%d] requesting to apply wishlist.", s.CharacterId())
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
			l.Infof("Character [%d] requesting purchase record for [%d].", s.CharacterId(), sp.SerialNumber())
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

		com, err := commodity.NewProcessor(l, ctx).GetById(sp.SerialNumber())
		if err != nil {
			l.WithError(err).Errorf("Unable to resolve commodity [%d] to a template id for character [%d] name change request.", sp.SerialNumber(), characterId)
			announceCashShopRejection(l, ctx, wp)(s, "Unable to process your name change request.")
			return
		}

		rm, err := pendingchange.NewProcessor(l, ctx).RequestNameChange(characterId, sp.NewName(), com.ItemId)
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
		if err := cashshop.NewProcessor(l, ctx).RequestPurchase(characterId, sp.SerialNumber(), false, 0, 0, transactionId); err != nil {
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
// rejections use it instead of pink text.
func handleBuyWorldTransfer(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *cashsb.ShopOperationBuyWorldTransfer) {
	return func(s session.Model, sp *cashsb.ShopOperationBuyWorldTransfer) {
		characterId := s.CharacterId()

		com, err := commodity.NewProcessor(l, ctx).GetById(sp.SerialNumber())
		if err != nil {
			l.WithError(err).Errorf("Unable to resolve commodity [%d] to a template id for character [%d] world transfer request.", sp.SerialNumber(), characterId)
			announceTransferWorldFailure(l, ctx, wp)(s, "unknown_error")
			return
		}

		destinationWorldId := world.Id(sp.TargetWorld())
		rm, err := pendingchange.NewProcessor(l, ctx).RequestWorldTransfer(characterId, destinationWorldId, com.ItemId)
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
		if err := cashshop.NewProcessor(l, ctx).RequestPurchase(characterId, sp.SerialNumber(), false, 0, 0, transactionId); err != nil {
			l.WithError(err).Errorf("Unable to request purchase for character [%d] serial number [%d] transaction [%s]; pending world transfer record [%s] may be orphaned.", characterId, sp.SerialNumber(), transactionId, rm.Id)
		}
	}
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

// announceTransferWorldFailure emits the real TRANSFER_WORLD_FAILED arm.
// reason is a key into the tenant's "errors" code table (same convention as
// every other *_FAILED body in this family, e.g. CashShopBuyFailedBody),
// not display text.
func announceTransferWorldFailure(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, reason string) {
	return func(s session.Model, reason string) {
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
