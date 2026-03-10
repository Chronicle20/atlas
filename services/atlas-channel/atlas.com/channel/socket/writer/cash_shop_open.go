package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"context"

	cashpkt "github.com/Chronicle20/atlas-packet/cash"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOpen = "CashShopOpen"

func CashShopOpenBody(a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			// Encode character info using the existing WriteCharacterInfo encoder.
			ciw := response.NewWriter(l)
			WriteCharacterInfo(l, ctx, options)(ciw)(c, bl)
			characterInfoBytes := ciw.Bytes()

			return cashpkt.NewCashShopOpen(characterInfoBytes, a.Name()).Encode(l, ctx)(options)
		}
	}
}
