package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	NPCShopOperation                       = "NPCShopOperation"
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
			w := response.NewWriter(l)
			w.WriteByte(getNpcShopOperation(l)(options, code))
			return w.Bytes()
		}
	}
}

func NPCShopOperationGenericErrorBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getNpcShopOperation(l)(options, NPCShopOperationGenericError))
			w.WriteBool(false)
			return w.Bytes()
		}
	}
}

func NPCShopOperationGenericErrorWithReasonBody(reason string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getNpcShopOperation(l)(options, NPCShopOperationGenericErrorWithReason))
			w.WriteBool(true)
			w.WriteAsciiString(reason)
			return w.Bytes()
		}
	}
}

func NPCShopOperationOverLevelRequirementBody(levelLimit uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getNpcShopOperation(l)(options, NPCShopOperationOverLevelRequirement))
			w.WriteInt(levelLimit)
			return w.Bytes()
		}
	}
}

func NPCShopOperationUnderLevelRequirementBody(levelLimit uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getNpcShopOperation(l)(options, NPCShopOperationUnderLevelRequirement))
			w.WriteInt(levelLimit)
			return w.Bytes()
		}
	}
}

func getNpcShopOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
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
