package writer

import (
	"github.com/Chronicle20/atlas-socket/packet"

	cashpkt "github.com/Chronicle20/atlas-packet/cash"
)

const CashShopCashQueryResult = "CashShopCashQueryResult"

func CashShopCashQueryResultBody(credit uint32, points uint32, prepaid uint32) packet.Encode {
	return cashpkt.NewCashQueryResult(credit, points, prepaid).Encode
}
