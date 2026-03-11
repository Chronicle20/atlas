package writer

import (
	"atlas-channel/account"
	asset2 "atlas-channel/asset"
	"atlas-channel/cashshop/inventory/asset"
	"atlas-channel/cashshop/wishlist"
	model2 "atlas-channel/socket/model"
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	cashpkt "github.com/Chronicle20/atlas-packet/cash"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
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

func CashShopCashInventoryBody(a account.Model, characterId uint32, assets []asset.Model, storageSlots uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCashShopOperation(l)(options, CashShopOperationLoadInventorySuccess)
			items := make([]cashpkt.CashInventoryItem, len(assets))
			for i, as := range assets {
				items[i] = cashpkt.CashInventoryItem{
					CashId:      as.Item().CashId(),
					AccountId:   a.Id(),
					CharacterId: characterId,
					TemplateId:  as.Item().TemplateId(),
					CommodityId: as.CommodityId(),
					Quantity:    int16(as.Item().Quantity()),
					GiftFrom:    "",
					Expiration:  msTime(as.Expiration()),
				}
			}
			return cashpkt.NewCashShopInventory(mode, items, storageSlots, a.CharacterSlots()).Encode(l, ctx)(options)
		}
	}
}

func CashShopCashInventoryPurchaseSuccessBody(accountId uint32, characterId uint32, a asset.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCashShopOperation(l)(options, CashShopOperationPurchaseSuccess)
			item := cashpkt.CashInventoryItem{
				CashId:      a.Item().CashId(),
				AccountId:   accountId,
				CharacterId: characterId,
				TemplateId:  a.Item().TemplateId(),
				CommodityId: a.CommodityId(),
				Quantity:    int16(a.Item().Quantity()),
				GiftFrom:    "",
				Expiration:  msTime(a.Expiration()),
			}
			return cashpkt.NewCashShopPurchaseSuccess(mode, item).Encode(l, ctx)(options)
		}
	}
}

func CashShopCashGiftsBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			// TODO map codes for JMS — currently hardcoded to 0x4D
			return cashpkt.NewCashShopGifts(0x4D).Encode(l, ctx)(options)
		}
	}
}

func CashShopInventoryCapacityIncreaseSuccessBody(inventoryType byte, capacity uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCashShopOperation(l)(options, CashShopOperationInventoryCapacityIncreaseSuccess)
			return cashpkt.NewInventoryCapacitySuccess(mode, inventoryType, uint16(capacity)).Encode(l, ctx)(options)
		}
	}
}

func CashShopInventoryCapacityIncreaseFailedBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCashShopOperation(l)(options, CashShopOperationInventoryCapacityIncreaseFailed)
			errorCode := getCashShopOperationError(l)(options, message)
			return cashpkt.NewInventoryCapacityFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopWishListBody(update bool, items []wishlist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			var mode byte
			if update {
				mode = getCashShopOperation(l)(options, CashShopOperationUpdateWishlist)
			} else {
				mode = getCashShopOperation(l)(options, CashShopOperationLoadWishlist)
			}
			var sns []uint32
			for _, item := range items {
				sns = append(sns, item.SerialNumber())
			}
			return cashpkt.NewWishList(mode, sns).Encode(l, ctx)(options)
		}
	}
}

func CashShopCashItemMovedToInventoryBody(a asset2.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCashShopOperation(l)(options, CashShopOperationCashItemMovedToInventory)
			am := model2.NewAsset(true, a)
			return cashpkt.NewCashItemMovedToInventory(mode, uint16(a.Slot()), am).Encode(l, ctx)(options)
		}
	}
}

func CashShopCashItemMovedToCashInventoryBody(accountId uint32, characterId uint32, a asset.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getCashShopOperation(l)(options, CashShopOperationCashItemMovedToCashInventory)
			item := cashpkt.CashInventoryItem{
				CashId:      a.Item().CashId(),
				AccountId:   accountId,
				CharacterId: characterId,
				TemplateId:  a.Item().TemplateId(),
				CommodityId: a.CommodityId(),
				Quantity:    int16(a.Item().Quantity()),
				GiftFrom:    "",
				Expiration:  msTime(a.Expiration()),
			}
			return cashpkt.NewCashItemMovedToCashInventory(mode, item).Encode(l, ctx)(options)
		}
	}
}

func getCashShopOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		return atlas_packet.ResolveCode(l, options, "operations", key)
	}
}

func getCashShopOperationError(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		return atlas_packet.ResolveCode(l, options, "errors", key)
	}
}
