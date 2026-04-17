package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
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
			_ = cashshop.NewProcessor(l, ctx).RequestPurchase(s.CharacterId(), sp.SerialNumber(), sp.IsPoints(), sp.Currency(), sp.Zero())
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
			err = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(cashcb.CashShopWishListBody(true, sns))(s)
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
			l.Infof("Character [%d] requesting purchase name change from [%s] to [%s] via item [%d].", s.CharacterId(), sp.OldName(), sp.NewName(), sp.SerialNumber())
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyWorldTransfer) {
			sp := &cashsb.ShopOperationBuyWorldTransfer{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Infof("Character [%d] requesting purchase world transfer for [%d] via item [%d].", s.CharacterId(), sp.TargetWorld(), sp.SerialNumber())
			return
		}
		l.Warnf("Unhandled Cash Shop Operation [%d] issued by character [%d].", op, s.CharacterId())
	}
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
