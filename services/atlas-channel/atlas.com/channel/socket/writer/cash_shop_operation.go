package writer

import (
	"atlas-channel/account"
	asset2 "atlas-channel/asset"
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/wishlist"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	CashShopOperation                                 = "CashShopOperation"
	CashShopOperationLoadInventorySuccess             = "LOAD_INVENTORY_SUCCESS"
	CashShopOperationLoadInventoryFailure             = "LOAD_INVENTORY_FAILURE"
	CashShopOperationInventoryCapacityIncreaseSuccess = "INVENTORY_CAPACITY_INCREASE_SUCCESS"
	CashShopOperationInventoryCapacityIncreaseFailed  = "INVENTORY_CAPACITY_INCREASE_FAILED"
	CashShopOperationLoadWishlist                     = "LOAD_WISHLIST"
	CashShopOperationUpdateWishlist                   = "UPDATE_WISHLIST"
	CashShopOperationPurchaseSuccess                  = "PURCHASE_SUCCESS"
	CashShopOperationCashItemMovedToInventory         = "CASH_ITEM_MOVED_TO_INVENTORY"
	CashShopOperationCashItemMovedToCashInventory     = "CASH_ITEM_MOVED_TO_CASH_INVENTORY"

	CashShopOperationErrorUnknown                           = "UNKNOWN_ERROR"                         // 0x00
	CashShopOperationErrorRequestTimedOut                   = "REQUEST_TIMED_OUT"                     // 0xA3
	CashShopOperationErrorNotEnoughCash                     = "NOT_ENOUGH_CASH"                       // 0xA5
	CashShopOperationErrorCannotGiftWhenUnderage            = "CANNOT_GIFT_WHEN_UNDERAGE"             // 0xA6
	CashShopOperationErrorExceededGiftLimit                 = "EXCEEDED_GIFT_LIMIT"                   // 0xA7
	CashShopOperationErrorCannotGiftToOwnAccount            = "CANNOT_GIFT_TO_OWN_ACCOUNT"            // 0xA8
	CashShopOperationErrorIncorrectName                     = "INCORRECT_NAME"                        // 0xA9
	CashShopOperationErrorCannotGiftGenderRestriction       = "CANNOT_GIFT_GENDER_RESTRICTION"        // 0xAA
	CashShopOperationErrorCannotGiftRecipientInventoryFull  = "CANNOT_GIFT_RECIPIENT_INVENTORY_FULL"  // 0xAB
	CashShopOperationErrorExceededCashItemLimit             = "EXCEEDED_CASH_ITEM_LIMIT"              // 0xAC
	CashShopOperationErrorIncorrectNameOrGenderRestriction  = "INCORRECT_NAME_OR_GENDER_RESTRICTION"  // 0xAD
	CashShopOperationErrorInvalidCouponCode                 = "INVALID_COUPON_COUPON"                 // 0xB0
	CashShopOperationErrorCouponExpired                     = "COUPON_EXPIRED"                        // 0xB2
	CashShopOperationErrorCouponAlreadyUsed                 = "COUPON_ALREADY_USED"                   // 0xB3
	CashShopOperationErrorCouponInternetCafeRestriction     = "COUPON_INTERNET_CAFE_RESTRICTION"      // 0xB4
	CashShopOperationErrorInternetCafeCouponAlreadyUsed     = "INTERNET_CAFE_COUPON_ALREADY_USED"     // 0xB5
	CashShopOperationErrorInternetCafeCouponExpired         = "INTERNET_CAFE_COUPON_EXPIRED"          // 0xB6
	CashShopOperationErrorCouponNotRegistered               = "COUPON_NOT_REGISTERED"                 // 0xB7
	CashShopOperationErrorCouponGenderRestriction           = "COUPON_GENDER_RESTRICTION"             // 0xB8
	CashShopOperationErrorCouponCannotBeGifted              = "COUPON_CANNOT_BE_GIFTED"               // 0xB9
	CashShopOperationErrorCouponOnlyForMapleStory           = "COUPON_ONLY_FOR_MAPLE_STORY"           // 0xBA
	CashShopOperationErrorInventoryFull                     = "INVENTORY_FULL"                        // 0xBB
	CashShopOperationErrorNotAvailableForPurchase           = "NOT_AVAILABLE_FOR_PURCHASE"            // 0xBC
	CashShopOperationErrorCannotGiftInvalidNameOrGender     = "CANNOT_GIFT_INVALID_NAME_OR_GENDER"    // 0xBD
	CashShopOperationErrorCheckNameOfReceiver               = "CHECK_NAME_OF_RECEIVER"                // 0xBE
	CashShopOperationErrorNotAvailableForPurchaseAtThisHour = "NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR"    // 0xBF
	CashShopOperationErrorOutOfStock                        = "OUT_OF_STOCK"                          // 0xC0
	CashShopOperationErrorExceededSpendingLimit             = "EXCEEDED_SPENDING_LIMIT"               // 0xC1
	CashShopOperationErrorNotEnoughMesos                    = "NOT_ENOUGH_MESOS"                      // 0xC2
	CashShopOperationErrorCashShopNotAvailableDuringBeta    = "CASH_SHOP_NOT_AVAILABLE_DURING_BETA"   // 0xC3
	CashShopOperationErrorInvalidBirthday                   = "INVALID_BIRTHDAY"                      // 0xC4
	CashShopOperationErrorOnlyAvailableToUsersBuying        = "ONLY_AVAILABLE_TO_USERS_BUYING"        // 0xC7
	CashShopOperationErrorAlreadyApplied                    = "ALREADY_APPLIED"                       // 0xC8
	CashShopOperationErrorDailyPurchaseLimit                = "DAILY_PURCHASE_LIMIT"                  // 0xCD
	CashShopOperationErrorCouponUsageLimit                  = "COUPON_USAGE_LIMIT"                    // 0xD0
	CashShopOperationErrorCouponSystemAvailableSoon         = "COUPON_SYSTEM_AVAILABLE_SOON"          // 0xD2
	CashShopOperationErrorFifteenDayLimit                   = "FIFTEEN_DAY_LIMIT"                     // 0xD3
	CashShopOperationErrorNotEnoughGiftTokens               = "NOT_ENOUGH_GIFT_TOKENS"                // 0xD4
	CashShopOperationErrorCannotSendTechnicalDifficulties   = "CANNOT_SEND_TECHNICAL_DIFFICULTIES"    // 0xD5
	CashShopOperationErrorCannotGiftAccountAge              = "CANNOT_GIFT_ACCOUNT_AGE"               // 0xD6
	CashShopOperationErrorCannotGiftPreviousInfractions     = "CANNOT_GIFT_PREVIOUS_INFRACTIONS"      // 0xD7
	CashShopOperationErrorCannotGiftAtThisTime              = "CANNOT_GIFT_AT_THIS_TIME"              // 0xD8
	CashShopOperationErrorCannotGiftLimit                   = "CANNOT_GIFT_LIMIT"                     // 0xD9
	CashShopOperationErrorCannotGiftTechnicalDifficulties   = "CANNOT_GIFT_TECHNICAL_DIFFICULTIES"    // 0xDA
	CashShopOperationErrorCannotTransferUnderLevelTwenty    = "CANNOT_TRANSFER_UNDER_LEVEL_TWENTY"    // 0xDB
	CashShopOperationErrorCannotTransferToSameWorld         = "CANNOT_TRANSFER_TO_SAME_WORLD"         // 0xDC
	CashShopOperationErrorCannotTransferToNewWorld          = "CANNOT_TRANSFER_TO_NEW_WORLD"          // 0xDD
	CashShopOperationErrorCannotTransferOut                 = "CANNOT_TRANSFER_OUT"                   // 0xDE
	CashShopOperationErrorCannotTransferNoEmptySlots        = "CANNOT_TRANSFER_NO_EMPTY_SLOTS"        // 0xDF
	CashShopOperationErrorEventEndedOrCannotBeFreelyTested  = "EVENT_ENDED_OR_CANT_BE_FREELY_TESTED"  // 0xE0
	CashShopOperationErrorCannotBePurchasedWithMaplePoints  = "CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS" // 0xE6
	CashShopOperationErrorPleaseTryAgain                    = "PLEASE_TRY_AGAIN"                      // 0xE7
	CashShopOperationErrorCannotBePurchasedWhenUnderSeven   = "CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN"  // 0xE8
	CashShopOperationErrorCannotBeReceivedWhenUnderSeven    = "CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN"   // 0xE9

)

func CashShopCashInventoryBody(l logrus.FieldLogger) func(a account.Model, characterId uint32, assets []asset.Model, storageSlots uint16) BodyProducer {
	return func(a account.Model, characterId uint32, assets []asset.Model, storageSlots uint16) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCashShopOperation(l)(options, CashShopOperationLoadInventorySuccess))
			w.WriteShort(uint16(len(assets)))
			for _, i := range assets {
				_ = WriteCashInventoryItem(a.Id(), characterId, i)(w)
			}
			w.WriteShort(storageSlots)
			w.WriteInt16(a.CharacterSlots())
			return w.Bytes()
		}
	}
}

func CashShopCashInventoryPurchaseSuccessBody(l logrus.FieldLogger) func(accountId uint32, characterId uint32, asset asset.Model) BodyProducer {
	return func(accountId uint32, characterId uint32, asset asset.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCashShopOperation(l)(options, CashShopOperationPurchaseSuccess))
			_ = WriteCashInventoryItem(accountId, characterId, asset)(w)
			return w.Bytes()
		}
	}
}

func WriteCashInventoryItem(accountId uint32, characterId uint32, a asset.Model) model.Operator[*response.Writer] {
	return func(w *response.Writer) error {
		w.WriteInt64(a.Item().CashId())
		w.WriteInt(accountId)
		w.WriteInt(characterId)
		w.WriteInt(a.Item().TemplateId())
		w.WriteInt(a.CommodityId())
		w.WriteInt16(int16(a.Item().Quantity()))
		WritePaddedString(w, "", 13) // TODO
		w.WriteInt64(msTime(a.Expiration()))
		w.WriteInt(0) // TODO
		w.WriteInt(0) // TODO
		return nil
	}
}

func CashShopCashGiftsBody(l logrus.FieldLogger) func(tenant tenant.Model) BodyProducer {
	return func(tenant tenant.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// TODO map codes for JMS
			w.WriteByte(0x4D)
			w.WriteShort(0)
			// TODO load gifts
			return w.Bytes()
		}
	}
}

func CashShopInventoryCapacityIncreaseSuccessBody(l logrus.FieldLogger) func(inventoryType byte, capacity uint32) BodyProducer {
	return func(inventoryType byte, capacity uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCashShopOperation(l)(options, CashShopOperationInventoryCapacityIncreaseSuccess))
			w.WriteByte(inventoryType)
			w.WriteShort(uint16(capacity))
			return w.Bytes()
		}
	}
}

func CashShopInventoryCapacityIncreaseFailedBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCashShopOperation(l)(options, CashShopOperationInventoryCapacityIncreaseFailed))
			w.WriteByte(getCashShopOperationError(l)(options, message))
			return w.Bytes()
		}
	}
}

func CashShopWishListBody(l logrus.FieldLogger) func(t tenant.Model) func(update bool, items []wishlist.Model) BodyProducer {
	return func(t tenant.Model) func(update bool, items []wishlist.Model) BodyProducer {
		return func(update bool, items []wishlist.Model) BodyProducer {
			return func(w *response.Writer, options map[string]interface{}) []byte {
				if update {
					w.WriteByte(getCashShopOperation(l)(options, CashShopOperationUpdateWishlist))
				} else {
					w.WriteByte(getCashShopOperation(l)(options, CashShopOperationLoadWishlist))
				}
				for _, item := range items {
					w.WriteInt(item.SerialNumber())
				}
				for i := 0; i < 10-len(items); i++ {
					w.WriteInt(0)
				}
				return w.Bytes()
			}
		}
	}
}

func CashShopCashItemMovedToInventoryBody(l logrus.FieldLogger, t tenant.Model) func(a asset2.Model[any]) BodyProducer {
	return func(a asset2.Model[any]) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCashShopOperation(l)(options, CashShopOperationCashItemMovedToInventory))
			w.WriteShort(uint16(a.Slot()))
			_ = WriteAssetInfo(t)(true)(w)(a)
			return w.Bytes()
		}
	}
}

func CashShopCashItemMovedToCashInventoryBody(l logrus.FieldLogger, t tenant.Model) func(accountId uint32, characterId uint32, a asset.Model) BodyProducer {
	return func(accountId uint32, characterId uint32, a asset.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCashShopOperation(l)(options, CashShopOperationCashItemMovedToCashInventory))
			_ = WriteCashInventoryItem(accountId, characterId, a)(w)
			return w.Bytes()
		}
	}
}

func getCashShopOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 99
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 99
		}
		return byte(res)
	}
}

func getCashShopOperationError(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["errors"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}
		return byte(res)
	}
}
