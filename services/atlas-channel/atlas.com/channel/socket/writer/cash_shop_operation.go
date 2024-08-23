package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopOperation = "CashShopOperation"

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

func CashShopWishListBody(l logrus.FieldLogger) func(tenant tenant.Model) BodyProducer {
	return func(tenant tenant.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// TODO map codes for JMS
			w.WriteByte(0x4F)
			for i := 0; i < 10; i++ {
				w.WriteInt(0)
			}
			return w.Bytes()
		}
	}
}
