package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	CashShopOperationHandle                = "CashShopOperationHandle"
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
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		op := r.ReadByte()
		var err error
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuy) {
			isPoints := r.ReadBool()
			currency := r.ReadUint32()
			serialNumber := r.ReadUint32()
			zero := r.ReadUint32()
			_ = cashshop.NewProcessor(l, ctx).RequestPurchase(s.CharacterId(), serialNumber, isPoints, currency, zero)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationGift) {
			birthday := r.ReadUint32()
			serialNumber := r.ReadUint32()
			name := r.ReadAsciiString()
			message := r.ReadAsciiString()
			l.Infof("Character [%d] gifting [%d] to [%s] with message [%s]. birthday [%d]", s.CharacterId(), serialNumber, name, message, birthday)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationSetWishlist) {
			serialNumbers := make([]uint32, 0)
			for range 10 {
				serialNumbers = append(serialNumbers, r.ReadUint32())
			}
			var wl []wishlist.Model
			wl, err = wishlist.NewProcessor(l, ctx).SetForCharacter(s.CharacterId(), serialNumbers)
			if err != nil {
				l.WithError(err).Errorf("Cash Shop Operation [%s] failed for character [%d].", CashShopOperationSetWishlist, s.CharacterId())
				return
			}
			err = session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopWishListBody(l)(t)(true, wl))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to update wish list for character [%d].", s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationIncreaseInventory) {
			isPoints := r.ReadBool()
			currency := r.ReadUint32()
			item := r.ReadBool()
			if !item {
				inventoryType := r.ReadByte()
				err = cashshop.NewProcessor(l, ctx).RequestInventoryIncreasePurchaseByType(s.CharacterId(), isPoints, currency, inventoryType)
				if err != nil {
					l.WithError(err).Errorf("Unable to request inventory increase purchase for character [%d].", s.CharacterId())
				}
			} else {
				serialNumber := r.ReadUint32()
				err = cashshop.NewProcessor(l, ctx).RequestInventoryIncreasePurchaseByItem(s.CharacterId(), isPoints, currency, serialNumber)
				if err != nil {
					l.WithError(err).Errorf("Unable to request inventory increase purchase for character [%d].", s.CharacterId())
				}
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationIncreaseStorage) {
			isPoints := r.ReadBool()
			currency := r.ReadUint32()
			item := r.ReadBool()
			if !item {
				err = cashshop.NewProcessor(l, ctx).RequestStorageIncreasePurchase(s.CharacterId(), isPoints, currency)
				if err != nil {
					l.WithError(err).Errorf("Unable to request storage increase purchase for character [%d].", s.CharacterId())
				}
			} else {
				serialNumber := r.ReadUint32()
				err = cashshop.NewProcessor(l, ctx).RequestStorageIncreasePurchaseByItem(s.CharacterId(), isPoints, currency, serialNumber)
				if err != nil {
					l.WithError(err).Errorf("Unable to request storage increase purchase for character [%d].", s.CharacterId())
				}
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationIncreaseCharacterSlot) {
			isPoints := r.ReadBool()
			currency := r.ReadUint32()
			serialNumber := r.ReadUint32()
			err = cashshop.NewProcessor(l, ctx).RequestCharacterSlotIncreasePurchaseByItem(s.CharacterId(), isPoints, currency, serialNumber)
			if err != nil {
				l.WithError(err).Errorf("Unable to request character slot increase purchase for character [%d].", s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationEnableEquipSlot) {
			pt := cashshop.GetPointType(r.ReadBool())
			serialNumber := r.ReadUint32()
			l.Infof("Character [%d] enabling equip slot? via item [%d] using [%s].", s.CharacterId(), serialNumber, pt)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationMoveFromCashInventory) {
			serialNumber := r.ReadUint64()
			inventoryType := r.ReadByte()
			slot := r.ReadInt16()
			err = cashshop.NewProcessor(l, ctx).MoveFromCashInventory(s.CharacterId(), serialNumber, inventoryType, slot)
			if err != nil {
				l.WithError(err).Errorf("Unable to move item [%d] from cash inventory to inventory [%d] slot [%d] for character [%d].", serialNumber, inventoryType, slot, s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationMoveToCashInventory) {
			serialNumber := r.ReadUint64()
			inventoryType := r.ReadByte()
			err = cashshop.NewProcessor(l, ctx).MoveToCashInventory(s.CharacterId(), serialNumber, inventoryType)
			if err != nil {
				l.WithError(err).Errorf("Unable to move item [%d] from inventory [%d] to cash inventory for character [%d].", serialNumber, inventoryType, s.CharacterId())
			}
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyNormal) {
			serialNumber := r.ReadUint32()
			l.Infof("Character [%d] purchasing [%d].", s.CharacterId(), serialNumber)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationRebateLockerItem) {
			birthday := r.ReadUint32()
			unk := r.ReadUint64()
			l.Infof("Character [%d] using rebate [%d]. birthday [%d]", s.CharacterId(), unk, birthday)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyCouple) {
			birthday := r.ReadUint32()
			option := r.ReadUint32()
			serialNumber := r.ReadUint32()
			name := r.ReadAsciiString()
			message := r.ReadAsciiString()
			l.Infof("Character [%d] purchasing [%d] for [%s] with message [%s]. Option [%d], birthday [%d]", s.CharacterId(), serialNumber, name, message, option, birthday)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyPackage) {
			pt := cashshop.GetPointType(r.ReadBool())
			option := r.ReadUint32()
			serialNumber := r.ReadUint32()
			l.Infof("Character [%d] purchasing [%d] with [%s]. Option [%d]", s.CharacterId(), serialNumber, pt, option)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationApplyWishlist) {
			l.Infof("Character [%d] requesting to apply wishlist.", s.CharacterId())
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyFriendship) {
			birthday := r.ReadUint32()
			option := r.ReadUint32()
			serialNumber := r.ReadUint32()
			name := r.ReadAsciiString()
			message := r.ReadAsciiString()
			l.Infof("Character [%d] purchasing [%d] for [%s] with message [%s]. Option [%d], birthday [%d]", s.CharacterId(), serialNumber, name, message, option, birthday)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationGetPurchaseRecord) {
			serialNumber := r.ReadUint32()
			l.Infof("Character [%d] requesting purchase record for [%d].", s.CharacterId(), serialNumber)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyNameChange) {
			serialNumber := r.ReadUint32()
			oldName := r.ReadAsciiString()
			newName := r.ReadAsciiString()
			l.Infof("Character [%d] requesting purchase name change from [%s] to [%s] via item [%d].", s.CharacterId(), oldName, newName, serialNumber)
			return
		}
		if isCashShopOperation(l)(readerOptions, op, CashShopOperationBuyWorldTransfer) {
			serialNumber := r.ReadUint32()
			targetWorld := r.ReadUint32()
			l.Infof("Character [%d] requesting purchase world transfer for [%d] via item [%d].", s.CharacterId(), targetWorld, serialNumber)
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
