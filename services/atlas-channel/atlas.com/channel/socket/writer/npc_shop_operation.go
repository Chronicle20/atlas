package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
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

func NPCShopOperationBody(l logrus.FieldLogger, t tenant.Model) func(code string) BodyProducer {
	return func(code string) BodyProducer {
		if code == NPCShopOperationOverLevelRequirement {
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationOverLevelRequirementBody(l, t)(200)
		} else if code == NPCShopOperationUnderLevelRequirement {
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationUnderLevelRequirementBody(l, t)(0)
		} else if code == NPCShopOperationGenericError {
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationGenericErrorBody(l, t)
		} else if code == NPCShopOperationGenericErrorWithReason {
			l.Warnf("Should be using non generic function for this code.")
			return NPCShopOperationGenericErrorWithReasonBody(l, t)("generic error")
		}
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getNpcShopOperation(l)(options, code))
			return w.Bytes()
		}
	}
}

func NPCShopOperationGenericErrorBody(l logrus.FieldLogger, _ tenant.Model) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getNpcShopOperation(l)(options, NPCShopOperationGenericError))
		w.WriteBool(false)
		return w.Bytes()
	}
}

func NPCShopOperationGenericErrorWithReasonBody(l logrus.FieldLogger, _ tenant.Model) func(reason string) BodyProducer {
	return func(reason string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getNpcShopOperation(l)(options, NPCShopOperationGenericErrorWithReason))
			w.WriteBool(true)
			w.WriteAsciiString(reason)
			return w.Bytes()
		}
	}
}

func NPCShopOperationOverLevelRequirementBody(l logrus.FieldLogger, _ tenant.Model) func(levelLimit uint32) BodyProducer {
	return func(levelLimit uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getNpcShopOperation(l)(options, NPCShopOperationOverLevelRequirement))
			w.WriteInt(levelLimit)
			return w.Bytes()
		}
	}
}

func NPCShopOperationUnderLevelRequirementBody(l logrus.FieldLogger, _ tenant.Model) func(levelLimit uint32) BodyProducer {
	return func(levelLimit uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
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
