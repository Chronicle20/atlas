package npc

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
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

func NPCShopOperationBody(code string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
		if code == NPCShopOperationOverLevelRequirement {
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationOverLevelRequirementBody(200)(l, ctx)
		} else if code == NPCShopOperationUnderLevelRequirement {
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationUnderLevelRequirementBody(0)(l, ctx)
		} else if code == NPCShopOperationGenericError {
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationGenericErrorBody()(l, ctx)
		} else if code == NPCShopOperationGenericErrorWithReason {
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationGenericErrorWithReasonBody("generic error")(l, ctx)
		}
		return atlas_packet.WithResolvedCode("operations", code, func(mode byte) packet.Encoder {
			return NewShopOperationSimple(mode)
		})(l, ctx)
	}
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
		return NewShopOperationLevelRequirement(mode, levelLimit)
	})
}

func NPCShopOperationUnderLevelRequirementBody(levelLimit uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationUnderLevelRequirement, func(mode byte) packet.Encoder {
		return NewShopOperationLevelRequirement(mode, levelLimit)
	})
}
