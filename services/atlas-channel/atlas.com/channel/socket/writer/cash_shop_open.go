package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/character/teleportrock"
	"atlas-channel/maps/location"
	"context"

	"github.com/sirupsen/logrus"

	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func CashShopOpenBody(a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			trm, err := teleportrock.NewProcessor(l, ctx).GetByCharacterId(c.Id())
			if err != nil {
				// Fail-open: a missing list must never block login (design §4.4).
				l.WithError(err).Warnf("Unable to fetch teleport-rock maps for character [%d]; sending empty lists.", c.Id())
				trm = teleportrock.Model{}
			}
			cd := BuildCharacterData(c, bl, location.ResolveMapId(l, ctx, c.Id()), trm)
			return cashpkt.NewCashShopOpen(cd, a.Name()).Encode(l, ctx)(options)
		}
	}
}
