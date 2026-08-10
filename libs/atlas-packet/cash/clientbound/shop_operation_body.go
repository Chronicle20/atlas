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

	// Gift/coupon/package item-blob arm operation keys (task-183 Wave 1.3).
	CashShopOperationGiftDone        = "GIFT_SUCCESS"
	CashShopOperationLoadGiftDone    = "LOAD_GIFT_SUCCESS"
	CashShopOperationUseCouponDone   = "USE_COUPON_SUCCESS"
	CashShopOperationGiftCouponDone  = "GIFT_COUPON_SUCCESS"
	CashShopOperationBuyPackageDone  = "BUY_PACKAGE_SUCCESS"
	CashShopOperationGiftPackageDone = "GIFT_PACKAGE_SUCCESS"
	CashShopOperationBuyNormalDone   = "BUY_NORMAL_SUCCESS"
	CashShopOperationFriendshipDone  = "FRIENDSHIP_SUCCESS"
	CashShopOperationRebateDone      = "REBATE_SUCCESS"
	CashShopOperationCoupleDone      = "COUPLE_SUCCESS"

	// Scalar/notice/transfer/gachapon/maple-point arm operation keys (task-183 Wave 1.4).
	CashShopOperationLimitGoodsCountChanged = "LIMIT_GOODS_COUNT_CHANGED"
	CashShopOperationDestroyDone            = "DESTROY_SUCCESS"
	CashShopOperationExpireDone             = "EXPIRE_DONE"
	CashShopOperationPurchaseRecordDone     = "PURCHASE_RECORD"
	CashShopOperationFreeCashItemDone       = "FREE_CASH_ITEM_DONE"
	CashShopOperationNameChangeBuyDone      = "NAME_CHANGE_BUY_DONE"
	CashShopOperationTransferWorldDone      = "TRANSFER_WORLD_SUCCESS"
	CashShopOperationGachaponOpenDone       = "GACHAPON_OPEN_SUCCESS"
	CashShopOperationGachaponCopyDone       = "GACHAPON_COPY_SUCCESS"
	CashShopOperationChangeMaplePointDone   = "CHANGE_MAPLE_POINT_SUCCESS"

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
	CashShopOperationErrorInvalidCouponCode                 = "INVALID_COUPON_CODE"
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

	// JMS-only arm operation keys (task-183 follow-up). These 10 arms exist only
	// in the JMS v185 dispatcher switch (see shop_operation_result_jms.go +
	// arm-catalog.md "JMS-only arms"). jms_v185 mode only; n-a on every GMS/legacy
	// version (absent from those switches).
	CashShopOperationGiftResultNotice          = "GIFT_RESULT_NOTICE"           // mode 76
	CashShopOperationLoadReceivedGiftSuccess   = "LOAD_RECEIVED_GIFT_SUCCESS"   // mode 77
	CashShopOperationLimitGoodsStockChanged    = "LIMIT_GOODS_STOCK_CHANGED"    // mode 96
	CashShopOperationShowNotice1089            = "SHOW_NOTICE_1089"             // mode 146 (bodyless)
	CashShopOperationTransferWorldNoticeReason = "TRANSFER_WORLD_NOTICE_REASON" // mode 147
	CashShopOperationRefreshLocker             = "REFRESH_LOCKER"               // mode 162
	CashShopOperationClientNoOp                = "CLIENT_NO_OP"                 // mode 164 (bodyless)
	CashShopOperationShowNotice1465            = "SHOW_NOTICE_1465"             // mode 166 (bodyless)
	CashShopOperationRefreshLockerOrNotice     = "REFRESH_LOCKER_OR_NOTICE"     // mode 167
	CashShopOperationShowNotice1464            = "SHOW_NOTICE_1464"             // mode 168 (bodyless)
)

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

// couponUnknownErrorDefaultByte is the wire byte NewUseCouponFailed sends for
// the UNKNOWN_ERROR case. UNKNOWN_ERROR is the client jump table's DEFAULT
// arm on every version, is intentionally absent from every template's
// "errors" table, and 99 is proven unmapped on all ten versions (pinned by
// TestCouponFailedUnknownErrorFallsThroughToTheDefaultNotice in
// kafka/consumer/cashshop/consumer_test.go, which asserts both reason==99
// and that 99 is not a reserved byte on any version). It is the SAME byte
// ResolveCode's generic miss-fallback would have produced; the only change
// here is skipping ResolveCode's ERROR-level "will likely cause a client
// crash" log, which is correct for an unconfigured opcode but misleading for
// this deliberately-unconfigured, non-crashing default-notice path.
const couponUnknownErrorDefaultByte byte = 99

func CashShopUseCouponFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationUseCouponFailed)
			if message == CashShopOperationErrorUnknown {
				// Short-circuit before ResolveCode's generic "errors" lookup.
				// This is an ordinary path (missing locker row, transaction
				// failure) that intentionally has no "errors" table entry, not
				// a misconfiguration — logging it at ERROR sends operators
				// chasing a client crash that will not happen.
				l.Debugf("Coupon failure [%s] has no configured error code; resolving to the client's default-notice byte (%d) — this is the intended fallback, not a misconfiguration.", message, couponUnknownErrorDefaultByte)
				return NewUseCouponFailed(mode, couponUnknownErrorDefaultByte).Encode(l, ctx)(options)
			}
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

// --- Gift/coupon/package item-blob arm bodies (task-183 Wave 1.3) ---
// See shop_operation_result_gift.go for the discrete structs and
// arm-catalog.md for the per-arm wire-truth. This wave resolves the legacy
// 0x4D gift TODO (CashShopCashGiftsBody / CashShopGifts stub) — GIFT_SUCCESS
// and LOAD_GIFT_SUCCESS below are their real RE'd replacements.

// CashShopGiftDoneBody builds the GIFT_SUCCESS arm (CCashShop::OnCashItemResGiftDone).
// Pure scalar body — no item-blob (task-0.3e report).
func CashShopGiftDoneBody(recipientName string, itemId int32, quantity uint16, nxCashSpent int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationGiftDone, func(mode byte) packet.Encoder {
		return NewGiftDone(mode, recipientName, itemId, quantity, nxCashSpent)
	})
}

// CashShopLoadGiftDoneBody builds the LOAD_GIFT_SUCCESS arm (CCashShop::OnCashItemResLoadGiftDone).
func CashShopLoadGiftDoneBody(gifts []GiftListEntry) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationLoadGiftDone, func(mode byte) packet.Encoder {
		return NewLoadGiftDone(mode, gifts)
	})
}

// CashShopUseCouponDoneBody builds the USE_COUPON_SUCCESS arm (CCashShop::OnCashItemResUseCouponDone).
func CashShopUseCouponDoneBody(items []CashInventoryItem, maplePoint int32, refs []PackedCashItemRef, meso int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationUseCouponDone, func(mode byte) packet.Encoder {
		return NewUseCouponDone(mode, items, maplePoint, refs, meso)
	})
}

// CashShopGiftCouponDoneBody builds the GIFT_COUPON_SUCCESS arm (CCashShop::OnCashItemResGiftCouponDone).
func CashShopGiftCouponDoneBody(recipientName string, items []CashInventoryItem, maplePoint int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationGiftCouponDone, func(mode byte) packet.Encoder {
		return NewGiftCouponDone(mode, recipientName, items, maplePoint)
	})
}

// CashShopBuyPackageDoneBody builds the BUY_PACKAGE_SUCCESS arm (CCashShop::OnCashItemResBuyPackageDone).
func CashShopBuyPackageDoneBody(items []CashInventoryItem, trailingCount uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationBuyPackageDone, func(mode byte) packet.Encoder {
		return NewBuyPackageDone(mode, items, trailingCount)
	})
}

// CashShopGiftPackageDoneBody builds the GIFT_PACKAGE_SUCCESS arm (CCashShop::OnCashItemResGiftPackageDone).
// Pure scalar body — no item-blob (task-0.3e report).
func CashShopGiftPackageDoneBody(recipientName string, packageId int32, unused1 uint16, unused2 uint16, nxCashSpent int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationGiftPackageDone, func(mode byte) packet.Encoder {
		return NewGiftPackageDone(mode, recipientName, packageId, unused1, unused2, nxCashSpent)
	})
}

// CashShopBuyNormalDoneBody builds the BUY_NORMAL_SUCCESS arm (CCashShop::OnCashItemResBuyNormalDone).
// List of PackedCashItemRef — no GW_CashItemInfo item-blob (task-0.3e/0.3f reports).
func CashShopBuyNormalDoneBody(refs []PackedCashItemRef) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationBuyNormalDone, func(mode byte) packet.Encoder {
		return NewBuyNormalDone(mode, refs)
	})
}

// CashShopFriendshipDoneBody builds the FRIENDSHIP_SUCCESS arm (CCashShop::OnCashItemResFriendShipDone).
func CashShopFriendshipDoneBody(item CashInventoryItem, recipientName string, itemId int32, quantity uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationFriendshipDone, func(mode byte) packet.Encoder {
		return NewFriendshipDone(mode, item, recipientName, itemId, quantity)
	})
}

// CashShopRebateDoneBody builds the REBATE_SUCCESS arm (CCashShop::OnCashItemResRebateDone).
// Pure scalar body — no item-blob (task-0.3e report).
func CashShopRebateDoneBody(sn int64, amount int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationRebateDone, func(mode byte) packet.Encoder {
		return NewRebateDone(mode, sn, amount)
	})
}

// CashShopCoupleDoneBody builds the COUPLE_SUCCESS arm (CCashShop::OnCashItemResCoupleDone).
func CashShopCoupleDoneBody(item CashInventoryItem, recipientName string, itemId int32, quantity uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationCoupleDone, func(mode byte) packet.Encoder {
		return NewCoupleDone(mode, item, recipientName, itemId, quantity)
	})
}

// --- Scalar/notice/transfer/gachapon/maple-point arm bodies (task-183 Wave 1.4) ---
// See shop_operation_result_misc.go / _transfer.go / _gachapon.go for the
// discrete structs and arm-catalog.md for the per-arm wire-truth.

// CashShopLimitGoodsCountChangedBody builds the LIMIT_GOODS_COUNT_CHANGED arm
// (CCashShop::OnCashItemResLimitGoodsCountChanged).
func CashShopLimitGoodsCountChangedBody(itemId int32, sn int32, remainCount int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationLimitGoodsCountChanged, func(mode byte) packet.Encoder {
		return NewLimitGoodsCountChanged(mode, itemId, sn, remainCount)
	})
}

// CashShopDestroyDoneBody builds the DESTROY_SUCCESS arm (CCashShop::OnCashItemResDestroyDone).
func CashShopDestroyDoneBody(sn int64) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationDestroyDone, func(mode byte) packet.Encoder {
		return NewDestroyDone(mode, sn)
	})
}

// CashShopExpireDoneBody builds the EXPIRE_DONE arm (CCashShop::OnCashItemResExpireDone).
func CashShopExpireDoneBody(sn int64) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationExpireDone, func(mode byte) packet.Encoder {
		return NewExpireDone(mode, sn)
	})
}

// CashShopPurchaseRecordDoneBody builds the PURCHASE_RECORD arm (CCashShop::OnCashItemResPurchaseRecord).
func CashShopPurchaseRecordDoneBody(goodsSN int32, purchased byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationPurchaseRecordDone, func(mode byte) packet.Encoder {
		return NewPurchaseRecordDone(mode, goodsSN, purchased)
	})
}

// CashShopFreeCashItemDoneBody builds the FREE_CASH_ITEM_DONE arm
// (CCashShop::OnCashItemResFreeCashItemDone). Item-blob body despite the
// catalog's "scalar" shape label (task-0.3d report).
func CashShopFreeCashItemDoneBody(item CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationFreeCashItemDone, func(mode byte) packet.Encoder {
		return NewFreeCashItemDone(mode, item)
	})
}

// CashShopNameChangeBuyDoneBody builds the NAME_CHANGE_BUY_DONE arm
// (CCashShop::OnCashItemNameChangeResBuyDone). Item-blob body despite the
// catalog's "scalar" shape label (task-0.3d report).
func CashShopNameChangeBuyDoneBody(item CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationNameChangeBuyDone, func(mode byte) packet.Encoder {
		return NewNameChangeBuyDone(mode, item)
	})
}

// CashShopTransferWorldDoneBody builds the TRANSFER_WORLD_SUCCESS arm
// (CCashShop::OnCashItemResTransferWorldDone). Item-blob body despite the
// catalog's "scalar" shape label (task-0.3d report).
func CashShopTransferWorldDoneBody(item CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationTransferWorldDone, func(mode byte) packet.Encoder {
		return NewTransferWorldDone(mode, item)
	})
}

// CashShopGachaponOpenDoneBody builds the GACHAPON_OPEN_SUCCESS arm
// (CCashShop::OnCashItemResCashGachaponOpenDone). Conditional item-blob,
// gated on isCashItem!=0 (task-0.3e report).
func CashShopGachaponOpenDoneBody(sn int64, remain int32, isCashItem byte, newItem CashInventoryItem, resultCode int32, resultParam2 byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationGachaponOpenDone, func(mode byte) packet.Encoder {
		return NewGachaponOpenDone(mode, sn, remain, isCashItem, newItem, resultCode, resultParam2)
	})
}

// CashShopGachaponCopyDoneBody builds the GACHAPON_COPY_SUCCESS arm
// (CCashShop::OnCashItemResCashGachaponCopyDone). Conditional item-blob,
// gated on flag1!=0 AND flag2!=0 (task-0.3e report).
func CashShopGachaponCopyDoneBody(flag1 byte, flag2 byte, unused1 int32, unused2 int32, lostItemId int32, lostNumber int32, item CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationGachaponCopyDone, func(mode byte) packet.Encoder {
		return NewGachaponCopyDone(mode, flag1, flag2, unused1, unused2, lostItemId, lostNumber, item)
	})
}

// CashShopChangeMaplePointDoneBody builds the CHANGE_MAPLE_POINT_SUCCESS arm
// (CCashShop::OnCashItemResChangeMaplePointDone).
func CashShopChangeMaplePointDoneBody(sn int64, count int32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationChangeMaplePointDone, func(mode byte) packet.Encoder {
		return NewChangeMaplePointDone(mode, sn, count)
	})
}

// --- JMS-only arm bodies (task-183 follow-up) ---
// See shop_operation_result_jms.go for the discrete structs and arm-catalog.md
// "JMS-only arms" for the per-arm wire-truth + behavior-derived-name evidence.
// Each body FIXES its operation key (the discrete struct never accepts a caller
// mode) and resolves the mode from the "operations" table. The two reason-notice
// arms (GIFT_RESULT_NOTICE, TRANSFER_WORLD_NOTICE_REASON) additionally resolve
// their reason byte from the writer "errors" table via `message` (the sibling
// failure-arm pattern; `message` flows into the "errors" key, never "operations"
// — allowed per AP-4/INV-3).

// CashShopGiftResultNoticeBody builds the GIFT_RESULT_NOTICE arm (mode 76,
// CCashShop__OnCashItemResShowGiftResultNotice @ 0x48ba24).
func CashShopGiftResultNoticeBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationGiftResultNotice)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewGiftResultNotice(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

// CashShopLoadReceivedGiftDoneBody builds the LOAD_RECEIVED_GIFT_SUCCESS arm
// (mode 77, CCashShop__OnCashItemResLoadReceivedGiftDone @ 0x48ba3f).
func CashShopLoadReceivedGiftDoneBody(flag byte, gifts []ReceivedGiftEntry) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationLoadReceivedGiftSuccess, func(mode byte) packet.Encoder {
		return NewLoadReceivedGiftDone(mode, flag, gifts)
	})
}

// CashShopLimitGoodsStockChangedBody builds the LIMIT_GOODS_STOCK_CHANGED arm
// (mode 96, CCashShop__OnCashItemResLimitGoodsStockChanged @ 0x48d4d4). `result`
// is a plain protocol status code that gates the conditional itemId field — see
// LimitGoodsStockChanged for why it is not config-resolved.
func CashShopLimitGoodsStockChangedBody(result byte, itemId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationLimitGoodsStockChanged, func(mode byte) packet.Encoder {
		return NewLimitGoodsStockChanged(mode, result, itemId)
	})
}

// CashShopShowNotice1089Body builds the bodyless SHOW_NOTICE_1089 arm (mode 146,
// CCashShop__OnCashItemResShowNotice1089 @ 0x48e6c9).
func CashShopShowNotice1089Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationShowNotice1089, func(mode byte) packet.Encoder {
		return NewShowNotice1089(mode)
	})
}

// CashShopTransferWorldNoticeReasonBody builds the TRANSFER_WORLD_NOTICE_REASON
// arm (mode 147, CCashShop__OnCashItemResTransferWorldNoticeReason @ 0x48e6f7).
func CashShopTransferWorldNoticeReasonBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", CashShopOperationTransferWorldNoticeReason)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", message)
			return NewTransferWorldNoticeReason(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

// CashShopRefreshLockerBody builds the REFRESH_LOCKER arm (mode 162,
// CCashShop__OnCashItemResRefreshLocker @ 0x48c321).
func CashShopRefreshLockerBody(item CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationRefreshLocker, func(mode byte) packet.Encoder {
		return NewRefreshLocker(mode, item)
	})
}

// CashShopClientNoOpBody builds the bodyless CLIENT_NO_OP arm (mode 164,
// nullsub_2 @ 0x48c370 — genuine client no-op).
func CashShopClientNoOpBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationClientNoOp, func(mode byte) packet.Encoder {
		return NewClientNoOp(mode)
	})
}

// CashShopShowNotice1465Body builds the bodyless SHOW_NOTICE_1465 arm (mode 166,
// CCashShop__OnCashItemResShowNotice1465 @ 0x48c26e).
func CashShopShowNotice1465Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationShowNotice1465, func(mode byte) packet.Encoder {
		return NewShowNotice1465(mode)
	})
}

// CashShopRefreshLockerOrNoticeBody builds the REFRESH_LOCKER_OR_NOTICE arm
// (mode 167, CCashShop__OnCashItemResRefreshLockerOrNotice @ 0x48c373).
func CashShopRefreshLockerOrNoticeBody(flag byte, item CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationRefreshLockerOrNotice, func(mode byte) packet.Encoder {
		return NewRefreshLockerOrNotice(mode, flag, item)
	})
}

// CashShopShowNotice1464Body builds the bodyless SHOW_NOTICE_1464 arm (mode 168,
// CCashShop__OnCashItemResShowNotice1464 @ 0x48c413).
func CashShopShowNotice1464Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationShowNotice1464, func(mode byte) packet.Encoder {
		return NewShowNotice1464(mode)
	})
}
