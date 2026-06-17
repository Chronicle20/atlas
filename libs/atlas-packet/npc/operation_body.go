package npc

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// CONFIRM_SHOP_TRANSACTION dispatcher (CShopDlg::OnPacket) per-mode body
// functions. Each wraps a verified per-mode codec in npc/clientbound and
// resolves its mode byte from the tenant template's "operations" table via the
// KEY string below. The supported arms are enumerated in
// docs/packets/dispatchers/npc_shop_operation.yaml.
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

// NPCShopOperationBody resolves a mode-only (Simple) notice arm by KEY string.
// The OVER/UNDER level-requirement, generic-error, and generic-error-with-reason
// arms carry wire data beyond the mode byte; callers should prefer the dedicated
// helpers below for those, but this keyed entry point keeps every supported arm
// reachable (falling back to sensible defaults for the data-bearing arms).
func NPCShopOperationBody(code string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
		switch code {
		case NPCShopOperationOverLevelRequirement:
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationOverLevelRequirementBody(200)(l, ctx)
		case NPCShopOperationUnderLevelRequirement:
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationUnderLevelRequirementBody(0)(l, ctx)
		case NPCShopOperationGenericError:
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationGenericErrorBody()(l, ctx)
		case NPCShopOperationGenericErrorWithReason:
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationGenericErrorWithReasonBody("generic error")(l, ctx)
		}
		return atlas_packet.WithResolvedCode("operations", code, func(mode byte) packet.Encoder {
			return clientbound.NewShopOperationSimple(mode)
		})(l, ctx)
	}
}

// NPCShopOperationGenericErrorBody is the GENERIC_ERROR arm with no reason
// string (hasReason = false).
func NPCShopOperationGenericErrorBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationGenericError, func(mode byte) packet.Encoder {
		return clientbound.NewShopOperationGenericError(mode)
	})
}

// NPCShopOperationGenericErrorWithReasonBody is the GENERIC_ERROR arm carrying a
// reason string (hasReason = true). Version-absent in jms_v185.
func NPCShopOperationGenericErrorWithReasonBody(reason string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationGenericErrorWithReason, func(mode byte) packet.Encoder {
		return clientbound.NewShopOperationGenericErrorWithReason(mode, reason)
	})
}

// NPCShopOperationOverLevelRequirementBody is the OVER_LEVEL_REQUIREMENT arm
// (mode + the level limit int).
func NPCShopOperationOverLevelRequirementBody(levelLimit uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationOverLevelRequirement, func(mode byte) packet.Encoder {
		return clientbound.NewShopOperationLevelRequirement(mode, levelLimit)
	})
}

// NPCShopOperationUnderLevelRequirementBody is the UNDER_LEVEL_REQUIREMENT arm
// (mode + the level limit int).
func NPCShopOperationUnderLevelRequirementBody(levelLimit uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NPCShopOperationUnderLevelRequirement, func(mode byte) packet.Encoder {
		return clientbound.NewShopOperationLevelRequirement(mode, levelLimit)
	})
}
