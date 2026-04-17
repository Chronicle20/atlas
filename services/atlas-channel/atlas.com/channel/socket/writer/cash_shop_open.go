package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"context"

	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func CashShopOpenBody(a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			cd := BuildCharacterData(c, bl)
			return cashpkt.NewCashShopOpen(cd, a.Name()).Encode(l, ctx)(options)
		}
	}
}
