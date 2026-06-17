package clientbound

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
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

	CashShopOperationErrorUnknown                           = "UNKNOWN_ERROR"
	CashShopOperationErrorRequestTimedOut                   = "REQUEST_TIMED_OUT"
	CashShopOperationErrorNotEnoughCash                     = "NOT_ENOUGH_CASH"
	CashShopOperationErrorCannotGiftWhenUnderage            = "CANNOT_GIFT_WHEN_UNDERAGE"
	CashShopOperationErrorExceededGiftLimit                 = "EXCEEDED_GIFT_LIMIT"
	CashShopOperationErrorCannotGiftToOwnAccount            = "CANNOT_GIFT_TO_OWN_ACCOUNT"
	CashShopOperationErrorIncorrectName                     = "INCORRECT_NAME"
	CashShopOperationErrorCannotGiftGenderRestriction       = "CANNOT_GIFT_GENDER_RESTRICTION"
	CashShopOperationErrorCannotGiftRecipientInventoryFull  = "CANNOT_GIFT_RECIPIENT_INVENTORY_FULL"
	CashShopOperationErrorExceededCashItemLimit             = "EXCEEDED_CASH_ITEM_LIMIT"
	CashShopOperationErrorIncorrectNameOrGenderRestriction  = "INCORRECT_NAME_OR_GENDER_RESTRICTION"
	CashShopOperationErrorInvalidCouponCode                 = "INVALID_COUPON_COUPON"
	CashShopOperationErrorCouponExpired                     = "COUPON_EXPIRED"
	CashShopOperationErrorCouponAlreadyUsed                 = "COUPON_ALREADY_USED"
	CashShopOperationErrorCouponInternetCafeRestriction     = "COUPON_INTERNET_CAFE_RESTRICTION"
	CashShopOperationErrorInternetCafeCouponAlreadyUsed     = "INTERNET_CAFE_COUPON_ALREADY_USED"
	CashShopOperationErrorInternetCafeCouponExpired         = "INTERNET_CAFE_COUPON_EXPIRED"
	CashShopOperationErrorCouponNotRegistered               = "COUPON_NOT_REGISTERED"
	CashShopOperationErrorCouponGenderRestriction           = "COUPON_GENDER_RESTRICTION"
	CashShopOperationErrorCouponCannotBeGifted              = "COUPON_CANNOT_BE_GIFTED"
	CashShopOperationErrorCouponOnlyForMapleStory           = "COUPON_ONLY_FOR_MAPLE_STORY"
	CashShopOperationErrorInventoryFull                     = "INVENTORY_FULL"
	CashShopOperationErrorNotAvailableForPurchase           = "NOT_AVAILABLE_FOR_PURCHASE"
	CashShopOperationErrorCannotGiftInvalidNameOrGender     = "CANNOT_GIFT_INVALID_NAME_OR_GENDER"
	CashShopOperationErrorCheckNameOfReceiver               = "CHECK_NAME_OF_RECEIVER"
	CashShopOperationErrorNotAvailableForPurchaseAtThisHour = "NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR"
	CashShopOperationErrorOutOfStock                        = "OUT_OF_STOCK"
	CashShopOperationErrorExceededSpendingLimit             = "EXCEEDED_SPENDING_LIMIT"
	CashShopOperationErrorNotEnoughMesos                    = "NOT_ENOUGH_MESOS"
	CashShopOperationErrorCashShopNotAvailableDuringBeta    = "CASH_SHOP_NOT_AVAILABLE_DURING_BETA"
	CashShopOperationErrorInvalidBirthday                   = "INVALID_BIRTHDAY"
	CashShopOperationErrorOnlyAvailableToUsersBuying        = "ONLY_AVAILABLE_TO_USERS_BUYING"
	CashShopOperationErrorAlreadyApplied                    = "ALREADY_APPLIED"
	CashShopOperationErrorDailyPurchaseLimit                = "DAILY_PURCHASE_LIMIT"
	CashShopOperationErrorCouponUsageLimit                  = "COUPON_USAGE_LIMIT"
	CashShopOperationErrorCouponSystemAvailableSoon         = "COUPON_SYSTEM_AVAILABLE_SOON"
	CashShopOperationErrorFifteenDayLimit                   = "FIFTEEN_DAY_LIMIT"
	CashShopOperationErrorNotEnoughGiftTokens               = "NOT_ENOUGH_GIFT_TOKENS"
	CashShopOperationErrorCannotSendTechnicalDifficulties   = "CANNOT_SEND_TECHNICAL_DIFFICULTIES"
	CashShopOperationErrorCannotGiftAccountAge              = "CANNOT_GIFT_ACCOUNT_AGE"
	CashShopOperationErrorCannotGiftPreviousInfractions     = "CANNOT_GIFT_PREVIOUS_INFRACTIONS"
	CashShopOperationErrorCannotGiftAtThisTime              = "CANNOT_GIFT_AT_THIS_TIME"
	CashShopOperationErrorCannotGiftLimit                   = "CANNOT_GIFT_LIMIT"
	CashShopOperationErrorCannotGiftTechnicalDifficulties   = "CANNOT_GIFT_TECHNICAL_DIFFICULTIES"
	CashShopOperationErrorCannotTransferUnderLevelTwenty    = "CANNOT_TRANSFER_UNDER_LEVEL_TWENTY"
	CashShopOperationErrorCannotTransferToSameWorld         = "CANNOT_TRANSFER_TO_SAME_WORLD"
	CashShopOperationErrorCannotTransferToNewWorld          = "CANNOT_TRANSFER_TO_NEW_WORLD"
	CashShopOperationErrorCannotTransferOut                 = "CANNOT_TRANSFER_OUT"
	CashShopOperationErrorCannotTransferNoEmptySlots        = "CANNOT_TRANSFER_NO_EMPTY_SLOTS"
	CashShopOperationErrorEventEndedOrCannotBeFreelyTested  = "EVENT_ENDED_OR_CANT_BE_FREELY_TESTED"
	CashShopOperationErrorCannotBePurchasedWithMaplePoints  = "CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS"
	CashShopOperationErrorPleaseTryAgain                    = "PLEASE_TRY_AGAIN"
	CashShopOperationErrorCannotBePurchasedWhenUnderSeven   = "CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN"
	CashShopOperationErrorCannotBeReceivedWhenUnderSeven    = "CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN"
)

func CashShopCashGiftsBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	// TODO map codes for JMS — currently hardcoded to 0x4D
	return NewCashShopGifts(0x4D).Encode
}

// CashShopLoadInventoryFailureBody builds the LOAD_INVENTORY_FAILURE arm
// (CCashShop::OnCashItemResLoadLockerFailed). It FIXES the LOAD_INVENTORY_FAILURE
// operation key (the discrete struct never accepts a caller mode) and resolves the
// reason byte from the writer's "errors" table.
func CashShopLoadInventoryFailureBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationLoadInventoryFailure)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewLoadInventoryFailure(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopInventoryCapacityIncreaseSuccessBody(inventoryType byte, capacity uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationInventoryCapacityIncreaseSuccess, func(mode byte) packet.Encoder {
		return NewInventoryCapacitySuccess(mode, inventoryType, uint16(capacity))
	})
}

func CashShopInventoryCapacityIncreaseFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationInventoryCapacityIncreaseFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewInventoryCapacityFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

// CashShopWishListLoadBody builds the LOAD_WISHLIST arm. It FIXES the
// LOAD_WISHLIST operation key (the discrete struct never accepts a caller mode).
func CashShopWishListLoadBody(sns []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationLoadWishlist, func(mode byte) packet.Encoder {
		return NewWishListLoad(mode, sns)
	})
}

// CashShopWishListUpdateBody builds the UPDATE_WISHLIST arm. It FIXES the
// UPDATE_WISHLIST operation key (the discrete struct never accepts a caller mode).
func CashShopWishListUpdateBody(sns []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationUpdateWishlist, func(mode byte) packet.Encoder {
		return NewWishListUpdate(mode, sns)
	})
}

func CashShopCashInventoryBody(items []CashInventoryItem, storageSlots uint16, characterSlots int16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationLoadInventorySuccess, func(mode byte) packet.Encoder {
		return NewCashShopInventory(mode, items, storageSlots, characterSlots)
	})
}

func CashShopCashInventoryPurchaseSuccessBody(item CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationPurchaseSuccess, func(mode byte) packet.Encoder {
		return NewCashShopPurchaseSuccess(mode, item)
	})
}

func CashShopCashItemMovedToInventoryBody(slot uint16, asset packetmodel.Asset) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationCashItemMovedToInventory, func(mode byte) packet.Encoder {
		return NewCashItemMovedToInventory(mode, slot, asset)
	})
}

func CashShopCashItemMovedToCashInventoryBody(item CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationCashItemMovedToCashInventory, func(mode byte) packet.Encoder {
		return NewCashItemMovedToCashInventory(mode, item)
	})
}
