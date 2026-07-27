package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
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

	// Failure-arm operation keys (task-183 Wave 1.1).
	CashShopOperationLoadGiftFailed              = "LOAD_GIFT_FAILED"
	CashShopOperationLoadWishFailed              = "LOAD_WISH_FAILED"
	CashShopOperationSetWishFailed               = "SET_WISH_FAILED"
	CashShopOperationBuyFailed                   = "BUY_FAILED"
	CashShopOperationUseCouponFailed             = "USE_COUPON_FAILED"
	CashShopOperationGiftFailed                  = "GIFT_FAILED"
	CashShopOperationIncTrunkCountFailed         = "INC_TRUNK_COUNT_FAILED"
	CashShopOperationIncCharacterSlotCountFailed = "INC_CHARACTER_SLOT_COUNT_FAILED"
	CashShopOperationIncBuyCharacterCountFailed  = "INC_BUY_CHARACTER_COUNT_FAILED"
	CashShopOperationEnableEquipSlotExtFailed    = "ENABLE_EQUIP_SLOT_EXT_FAILED"
	CashShopOperationMoveLToSFailed              = "MOVE_L_TO_S_FAILED"
	CashShopOperationMoveSToLFailed              = "MOVE_S_TO_L_FAILED"
	CashShopOperationDestroyFailed               = "DESTROY_FAILED"
	CashShopOperationRebateFailed                = "REBATE_FAILED"
	CashShopOperationCoupleFailed                = "COUPLE_FAILED"
	CashShopOperationBuyPackageFailed            = "BUY_PACKAGE_FAILED"
	CashShopOperationGiftPackageFailed           = "GIFT_PACKAGE_FAILED"
	CashShopOperationBuyNormalFailed             = "BUY_NORMAL_FAILED"
	CashShopOperationFriendshipFailed            = "FRIENDSHIP_FAILED"
	CashShopOperationPurchaseRecordFailed        = "PURCHASE_RECORD_FAILED"
	CashShopOperationTransferWorldFailed         = "TRANSFER_WORLD_FAILED"
	CashShopOperationGachaponOpenFailed          = "GACHAPON_OPEN_FAILED"
	CashShopOperationGachaponCopyFailed          = "GACHAPON_COPY_FAILED"
	CashShopOperationChangeMaplePointFailed      = "CHANGE_MAPLE_POINT_FAILED"

	// Counter-arm operation keys (task-183 Wave 1.2).
	CashShopOperationIncTrunkCountSuccess         = "INC_TRUNK_COUNT_SUCCESS"
	CashShopOperationIncCharacterSlotCountSuccess = "INC_CHARACTER_SLOT_COUNT_SUCCESS"
	CashShopOperationIncBuyCharacterCountSuccess  = "INC_BUY_CHARACTER_COUNT_SUCCESS"
	CashShopOperationEnableEquipSlotExtSuccess    = "ENABLE_EQUIP_SLOT_EXT_SUCCESS"

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

// --- Failure-arm bodies (task-183 Wave 1.1) ---
// Each mode+reason failure arm resolves its mode from the "operations" table
// (FIXED operation key, never caller-supplied) and its reason byte from the
// "errors" table via `message` (allowed per AP-4/INV-3).

func CashShopLoadGiftFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationLoadGiftFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewLoadGiftFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopLoadWishFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationLoadWishFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewLoadWishFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopSetWishFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationSetWishFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewSetWishFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

// CashShopBuyFailedBody builds the BUY_FAILED arm (CCashShop::OnCashItemResBuyFailed).
// FIXES the BUY_FAILED operation key; resolves the reason byte from the writer's
// "errors" table. `message` flows into the "errors" key (allowed), never "operations".
func CashShopBuyFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationBuyFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewBuyFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopUseCouponFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationUseCouponFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewUseCouponFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopGiftFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationGiftFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewGiftFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopIncTrunkCountFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationIncTrunkCountFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewIncTrunkCountFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopIncCharacterSlotCountFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationIncCharacterSlotCountFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewIncCharacterSlotCountFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopIncBuyCharacterCountFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationIncBuyCharacterCountFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewIncBuyCharacterCountFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopEnableEquipSlotExtFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationEnableEquipSlotExtFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewEnableEquipSlotExtFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopMoveLToSFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationMoveLToSFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewMoveLToSFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopMoveSToLFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationMoveSToLFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewMoveSToLFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopDestroyFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationDestroyFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewDestroyFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopRebateFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationRebateFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewRebateFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopCoupleFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationCoupleFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewCoupleFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopBuyPackageFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationBuyPackageFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewBuyPackageFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopGiftPackageFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationGiftPackageFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewGiftPackageFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopBuyNormalFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationBuyNormalFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewBuyNormalFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopFriendshipFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationFriendshipFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewFriendshipFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

// CashShopPurchaseRecordFailedBody builds the PURCHASE_RECORD_FAILED arm. The
// wire carries a reason byte that the client reads and discards (task-183
// design decision #3) — modeled as a normal mode+reason failure.
func CashShopPurchaseRecordFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationPurchaseRecordFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewPurchaseRecordFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopTransferWorldFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationTransferWorldFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewTransferWorldFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopGachaponOpenFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationGachaponOpenFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewGachaponOpenFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func CashShopGachaponCopyFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationGachaponCopyFailed)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewGachaponCopyFailed(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

// CashShopChangeMaplePointFailedBody builds the bodyless CHANGE_MAPLE_POINT_FAILED
// arm (mode byte only — RE-confirmed zero further reads). No "errors" resolution;
// no message/reason parameter (AP-4/INV-3: no caller-supplied op/code/mode param).
func CashShopChangeMaplePointFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationChangeMaplePointFailed, func(mode byte) packet.Encoder {
		return NewChangeMaplePointFailed(mode)
	})
}

// --- Counter-arm bodies (task-183 Wave 1.2) ---
// RE-proven shape: mode + uint16 absolute-counter update (NO inventory/slot-type
// byte). Each body func FIXES its operation key (the discrete struct never
// accepts a caller-supplied mode) and resolves the mode from the "operations"
// table.

// CashShopIncTrunkCountSuccessBody builds the INC_TRUNK_COUNT_SUCCESS arm
// (CCashShop::OnCashItemResIncTrunkCountDone).
func CashShopIncTrunkCountSuccessBody(trunkCount uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationIncTrunkCountSuccess, func(mode byte) packet.Encoder {
		return NewIncTrunkCountSuccess(mode, trunkCount)
	})
}

// CashShopIncCharacterSlotCountSuccessBody builds the INC_CHARACTER_SLOT_COUNT_SUCCESS
// arm (CCashShop::OnCashItemResIncCharacterSlotCountDone).
func CashShopIncCharacterSlotCountSuccessBody(slotCount uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationIncCharacterSlotCountSuccess, func(mode byte) packet.Encoder {
		return NewIncCharacterSlotCountSuccess(mode, slotCount)
	})
}

// CashShopIncBuyCharacterCountSuccessBody builds the INC_BUY_CHARACTER_COUNT_SUCCESS
// arm (CCashShop::OnCashItemResIncBuyCharacterCountDone). Present only in v95/jms
// among MODERN versions (n-a v83/v84/v87 per arm-catalog.md).
func CashShopIncBuyCharacterCountSuccessBody(buyCharacterCount uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationIncBuyCharacterCountSuccess, func(mode byte) packet.Encoder {
		return NewIncBuyCharacterCountSuccess(mode, buyCharacterCount)
	})
}

// CashShopEnableEquipSlotExtSuccessBody builds the ENABLE_EQUIP_SLOT_EXT_SUCCESS
// arm (CCashShop::OnCashItemResEnableEquipSlotExtDone). Wire is mode + TWO
// uint16 fields (slotIndex, days) — not a single count.
func CashShopEnableEquipSlotExtSuccessBody(slotIndex uint16, days uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationEnableEquipSlotExtSuccess, func(mode byte) packet.Encoder {
		return NewEnableEquipSlotExtSuccess(mode, slotIndex, days)
	})
}
