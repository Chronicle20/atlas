package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

const (
	NPCShopOperationOk                     = "OK"
	NPCShopOperationOutOfStock             = "OUT_OF_STOCK"
	NPCShopOperationNotEnoughMoney         = "NOT_ENOUGH_MONEY"
	NPCShopOperationInventoryFull          = "INVENTORY_FULL"
	NPCShopOperationOutOfStock2            = "OUT_OF_STOCK_2"
	NPCShopOperationOutOfStock3            = "OUT_OF_STOCK_3"
	NPCShopOperationNotEnoughMoney2        = "NOT_ENOUGH_MONEY_2"
	NPCShopOperationNeedMoreItems          = "NEED_MORE_ITEMS"
	NPCShopOperationOverLevelRequirement   = "OVER_LEVEL_REQUIREMENT"
	NPCShopOperationUnderLevelRequirement  = "UNDER_LEVEL_REQUIREMENT"
	NPCShopOperationTradeLimit             = "TRADE_LIMIT"
	NPCShopOperationGenericError           = "GENERIC_ERROR"
	NPCShopOperationGenericErrorWithReason = "GENERIC_ERROR_WITH_REASON"
)

// Per-mode body funcs. Each FIXES its own operation key (a hard-coded const)
// via WithResolvedCode and constructs that mode's DISCRETE struct. No body func
// accepts a caller-supplied op/code/mode — the version-resolved mode byte comes
// from the tenant template's "operations" table, keyed by the fixed const.

func NPCShopOperationOkBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationOk, func(mode byte) packet.Encoder {
		return NewShopOperationOk(mode)
	})
}

func NPCShopOperationOutOfStockBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationOutOfStock, func(mode byte) packet.Encoder {
		return NewShopOperationOutOfStock(mode)
	})
}

func NPCShopOperationNotEnoughMoneyBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationNotEnoughMoney, func(mode byte) packet.Encoder {
		return NewShopOperationNotEnoughMoney(mode)
	})
}

func NPCShopOperationInventoryFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationInventoryFull, func(mode byte) packet.Encoder {
		return NewShopOperationInventoryFull(mode)
	})
}

func NPCShopOperationOutOfStock2Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationOutOfStock2, func(mode byte) packet.Encoder {
		return NewShopOperationOutOfStock2(mode)
	})
}

func NPCShopOperationOutOfStock3Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationOutOfStock3, func(mode byte) packet.Encoder {
		return NewShopOperationOutOfStock3(mode)
	})
}

func NPCShopOperationNotEnoughMoney2Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationNotEnoughMoney2, func(mode byte) packet.Encoder {
		return NewShopOperationNotEnoughMoney2(mode)
	})
}

func NPCShopOperationNeedMoreItemsBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationNeedMoreItems, func(mode byte) packet.Encoder {
		return NewShopOperationNeedMoreItems(mode)
	})
}

func NPCShopOperationTradeLimitBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationTradeLimit, func(mode byte) packet.Encoder {
		return NewShopOperationTradeLimit(mode)
	})
}

func NPCShopOperationGenericErrorBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationGenericError, func(mode byte) packet.Encoder {
		return NewShopOperationGenericError(mode)
	})
}

func NPCShopOperationGenericErrorWithReasonBody(reason string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationGenericErrorWithReason, func(mode byte) packet.Encoder {
		return NewShopOperationGenericErrorWithReason(mode, reason)
	})
}

func NPCShopOperationOverLevelRequirementBody(levelLimit uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationOverLevelRequirement, func(mode byte) packet.Encoder {
		return NewShopOperationOverLevelRequirement(mode, levelLimit)
	})
}

func NPCShopOperationUnderLevelRequirementBody(levelLimit uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationUnderLevelRequirement, func(mode byte) packet.Encoder {
		return NewShopOperationUnderLevelRequirement(mode, levelLimit)
	})
}
