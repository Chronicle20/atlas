package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopCashQueryResult = "CashShopCashQueryResult"

func CashShopCashQueryResultBody(l logrus.FieldLogger) func(tenant tenant.Model) BodyProducer {
	return func(tenant tenant.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(0)
			w.WriteInt(0)
			if tenant.Region() == "GMS" && tenant.MajorVersion() > 12 {
				w.WriteInt(0)
			}
			return w.Bytes()
		}
	}
}
