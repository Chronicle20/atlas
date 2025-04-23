package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
)

const CashShopCashQueryResult = "CashShopCashQueryResult"

func CashShopCashQueryResultBody(tenant tenant.Model) func(credit uint32, points uint32, prepaid uint32) BodyProducer {
	return func(credit uint32, points uint32, prepaid uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(credit)
			w.WriteInt(points)
			if tenant.Region() == "GMS" && tenant.MajorVersion() > 12 {
				w.WriteInt(prepaid)
			}
			return w.Bytes()
		}
	}
}
