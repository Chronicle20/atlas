package writer

import (
	"context"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

func SetTamingMobInfoBody(characterId, level, exp, tiredness uint32, levelUp bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return charpkt.NewSetTamingMobInfo(characterId, level, exp, tiredness, levelUp).Encode(l, ctx)
	}
}
