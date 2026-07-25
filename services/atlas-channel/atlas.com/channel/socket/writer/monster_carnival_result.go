package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	carnivalpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MonsterCarnivalResultBody encodes the clientbound MONSTER_CARNIVAL_RESULT packet
// (CField_MonsterCarnival::OnShowGameResult), the end-of-match outcome.
// No emitter wires this writer yet; it is an intentional seam.
func MonsterCarnivalResultBody(result byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return carnivalpkt.NewMonsterCarnivalResult(result).Encode(l, ctx)(options)
		}
	}
}
