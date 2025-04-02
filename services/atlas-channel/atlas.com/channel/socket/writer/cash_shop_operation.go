package writer

import (
	"atlas-channel/cashshop/wishlist"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	CashShopOperation                                 = "CashShopOperation"
	CashShopOperationInventoryCapacityIncreaseSuccess = "INVENTORY_CAPACITY_INCREASE_SUCCESS"
	CashShopOperationInventoryCapacityIncreaseFailed  = "INVENTORY_CAPACITY_INCREASE_FAILED"
	CashShopOperationLoadWishlist                     = "LOAD_WISHLIST"
	CashShopOperationUpdateWishlist                   = "UPDATE_WISHLIST"
)

func CashShopCashInventoryBody(l logrus.FieldLogger) func(tenant tenant.Model) BodyProducer {
	return func(tenant tenant.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// TODO map codes for JMS
			w.WriteByte(0x4B)
			w.WriteShort(0)
			w.WriteShort(0) // character storage slots
			w.WriteInt16(4) // character slots
			return w.Bytes()
		}
	}
}

func CashShopCashGiftsBody(l logrus.FieldLogger) func(tenant tenant.Model) BodyProducer {
	return func(tenant tenant.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// TODO map codes for JMS
			w.WriteByte(0x4D)
			w.WriteShort(0)
			// TODO load gifts
			return w.Bytes()
		}
	}
}

func CashShopInventoryCapacityIncreaseSuccessBody(l logrus.FieldLogger) func(inventoryType byte, capacity uint32) BodyProducer {
	return func(inventoryType byte, capacity uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCashShopOperation(l)(options, CashShopOperationInventoryCapacityIncreaseSuccess))
			w.WriteByte(inventoryType)
			w.WriteShort(uint16(capacity))
			return w.Bytes()
		}
	}
}

func CashShopInventoryCapacityIncreaseFailedBody(l logrus.FieldLogger) func(message byte) BodyProducer {
	return func(message byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCashShopOperation(l)(options, CashShopOperationInventoryCapacityIncreaseFailed))
			w.WriteByte(message)
			return w.Bytes()
		}
	}
}

func CashShopWishListBody(l logrus.FieldLogger) func(t tenant.Model) func(update bool, items []wishlist.Model) BodyProducer {
	return func(t tenant.Model) func(update bool, items []wishlist.Model) BodyProducer {
		return func(update bool, items []wishlist.Model) BodyProducer {
			return func(w *response.Writer, options map[string]interface{}) []byte {
				if update {
					w.WriteByte(getCashShopOperation(l)(options, CashShopOperationUpdateWishlist))
				} else {
					w.WriteByte(getCashShopOperation(l)(options, CashShopOperationLoadWishlist))
				}
				for _, item := range items {
					w.WriteInt(item.SerialNumber())
				}
				for i := 0; i < 10-len(items); i++ {
					w.WriteInt(0)
				}
				return w.Bytes()
			}
		}
	}
}

func getCashShopOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
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
