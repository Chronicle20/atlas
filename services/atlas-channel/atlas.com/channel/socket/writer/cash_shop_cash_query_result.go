package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CashShopCashQueryResult = "CashShopCashQueryResult"

func CashShopCashQueryResultBody(credit uint32, points uint32, prepaid uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(credit)
			w.WriteInt(points)
			if t.Region() == "GMS" && t.MajorVersion() > 12 {
				w.WriteInt(prepaid)
			}
			return w.Bytes()
		}
	}
}
