package writer

import (
	"atlas-channel/account"
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/maps/location"
	"context"
	"errors"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func CashShopOpenBody(a account.Model, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mapId := _map.Id(0)
			if f, err := location.GetField(l, ctx, c.Id()); err == nil {
				mapId = f.MapId()
			} else if !errors.Is(err, location.ErrNotFound) {
				l.WithError(err).Warnf("Unable to resolve atlas-maps location for character [%d]; sending map 0 in CharacterData.", c.Id())
			}
			cd := BuildCharacterData(c, bl, mapId)
			return cashpkt.NewCashShopOpen(cd, a.Name()).Encode(l, ctx)(options)
		}
	}
}
