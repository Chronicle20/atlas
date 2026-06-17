package cash

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// CashShopOperation per-mode KEY strings. These are the keys in the tenant
// template's CashShopOperation writer "operations" table (see
// docs/packets/dispatchers/cash_shop_operation.yaml). Each maps to the leading
// Decode1 switch case in that version's CCashShop::OnCashItemResult.
const (
	CashShopOperationModeLoadInventorySuccess             = "LOAD_INVENTORY_SUCCESS"
	CashShopOperationModeLoadInventoryFailure             = "LOAD_INVENTORY_FAILURE"
	CashShopOperationModeLoadWishlist                     = "LOAD_WISHLIST"
	CashShopOperationModeUpdateWishlist                   = "UPDATE_WISHLIST"
	CashShopOperationModePurchaseSuccess                  = "PURCHASE_SUCCESS"
	CashShopOperationModeInventoryCapacityIncreaseSuccess = "INVENTORY_CAPACITY_INCREASE_SUCCESS"
	CashShopOperationModeInventoryCapacityIncreaseFailed  = "INVENTORY_CAPACITY_INCREASE_FAILED"
	CashShopOperationModeCashItemMovedToInventory         = "CASH_ITEM_MOVED_TO_INVENTORY"
	CashShopOperationModeCashItemMovedToCashInventory     = "CASH_ITEM_MOVED_TO_CASH_INVENTORY"
)

// CashShopOperationLoadInventorySuccessBody wraps the verified CashShopInventory
// codec (LOAD_INVENTORY_SUCCESS).
func CashShopOperationLoadInventorySuccessBody(items []clientbound.CashInventoryItem, storageSlots uint16, characterSlots int16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationModeLoadInventorySuccess, func(mode byte) packet.Encoder {
		return clientbound.NewCashShopInventory(mode, items, storageSlots, characterSlots)
	})
}

// CashShopOperationLoadInventoryFailureBody wraps the verified OperationError
// codec (LOAD_INVENTORY_FAILURE).
func CashShopOperationLoadInventoryFailureBody(errorCode byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationModeLoadInventoryFailure, func(mode byte) packet.Encoder {
		return clientbound.NewOperationError(mode, errorCode)
	})
}

// CashShopOperationLoadWishlistBody wraps the verified WishList codec
// (LOAD_WISHLIST).
func CashShopOperationLoadWishlistBody(items []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationModeLoadWishlist, func(mode byte) packet.Encoder {
		return clientbound.NewWishList(mode, items)
	})
}

// CashShopOperationUpdateWishlistBody wraps the verified WishList codec
// (UPDATE_WISHLIST).
func CashShopOperationUpdateWishlistBody(items []uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationModeUpdateWishlist, func(mode byte) packet.Encoder {
		return clientbound.NewWishList(mode, items)
	})
}

// CashShopOperationPurchaseSuccessBody wraps the verified CashShopPurchaseSuccess
// codec (PURCHASE_SUCCESS).
func CashShopOperationPurchaseSuccessBody(item clientbound.CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationModePurchaseSuccess, func(mode byte) packet.Encoder {
		return clientbound.NewCashShopPurchaseSuccess(mode, item)
	})
}

// CashShopOperationInventoryCapacityIncreaseSuccessBody wraps the verified
// InventoryCapacitySuccess codec (INVENTORY_CAPACITY_INCREASE_SUCCESS).
func CashShopOperationInventoryCapacityIncreaseSuccessBody(inventoryType byte, capacity uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationModeInventoryCapacityIncreaseSuccess, func(mode byte) packet.Encoder {
		return clientbound.NewInventoryCapacitySuccess(mode, inventoryType, capacity)
	})
}

// CashShopOperationInventoryCapacityIncreaseFailedBody wraps the verified
// InventoryCapacityFailed codec (INVENTORY_CAPACITY_INCREASE_FAILED).
func CashShopOperationInventoryCapacityIncreaseFailedBody(errorCode byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationModeInventoryCapacityIncreaseFailed, func(mode byte) packet.Encoder {
		return clientbound.NewInventoryCapacityFailed(mode, errorCode)
	})
}

// CashShopOperationCashItemMovedToInventoryBody wraps the verified
// CashItemMovedToInventory codec (CASH_ITEM_MOVED_TO_INVENTORY).
func CashShopOperationCashItemMovedToInventoryBody(slot uint16, asset model.Asset) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationModeCashItemMovedToInventory, func(mode byte) packet.Encoder {
		return clientbound.NewCashItemMovedToInventory(mode, slot, asset)
	})
}

// CashShopOperationCashItemMovedToCashInventoryBody wraps the verified
// CashItemMovedToCashInventory codec (CASH_ITEM_MOVED_TO_CASH_INVENTORY).
func CashShopOperationCashItemMovedToCashInventoryBody(item clientbound.CashInventoryItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CashShopOperationModeCashItemMovedToCashInventory, func(mode byte) packet.Encoder {
		return clientbound.NewCashItemMovedToCashInventory(mode, item)
	})
}
