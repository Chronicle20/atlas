package writer

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	npcpkt "github.com/Chronicle20/atlas-packet/npc"
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

func NPCShopOperationBody(code string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
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
		return func(options map[string]interface{}) []byte {
			mode := getNpcShopOperation(l)(options, code)
			return npcpkt.NewShopOperationSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func NPCShopOperationGenericErrorBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getNpcShopOperation(l)(options, NPCShopOperationGenericError)
			return npcpkt.NewShopOperationGenericError(mode).Encode(l, ctx)(options)
		}
	}
}

func NPCShopOperationGenericErrorWithReasonBody(reason string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getNpcShopOperation(l)(options, NPCShopOperationGenericErrorWithReason)
			return npcpkt.NewShopOperationGenericErrorWithReason(mode, reason).Encode(l, ctx)(options)
		}
	}
}

func NPCShopOperationOverLevelRequirementBody(levelLimit uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getNpcShopOperation(l)(options, NPCShopOperationOverLevelRequirement)
			return npcpkt.NewShopOperationLevelRequirement(mode, levelLimit).Encode(l, ctx)(options)
		}
	}
}

func NPCShopOperationUnderLevelRequirementBody(levelLimit uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getNpcShopOperation(l)(options, NPCShopOperationUnderLevelRequirement)
			return npcpkt.NewShopOperationLevelRequirement(mode, levelLimit).Encode(l, ctx)(options)
		}
	}
}

func getNpcShopOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		return atlas_packet.ResolveCode(l, options, "operations", key)
	}
}
