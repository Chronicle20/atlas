package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/effective_stats"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func CashShopOpenBody(channelId channel.Id, a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			maxHp, maxMp := effective_stats.ResolveCharacterMaxes(l, ctx, c.WorldId(), channelId, c.Id(), c.MaxHp(), c.MaxMp())
			cd := BuildCharacterData(c, bl, maxHp, maxMp)
			return cashpkt.NewCashShopOpen(cd, a.Name()).Encode(l, ctx)(options)
		}
	}
}
