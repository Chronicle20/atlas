package writer

import (
	"context"

	carnivalpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MonsterCarnivalObtainedCPBody encodes the clientbound MONSTER_CARNIVAL_OBTAINED_CP
// packet (CField_MonsterCarnival::OnPersonalCP), a personal CP scoreboard update.
// No emitter wires this writer yet; it is an intentional seam.
func MonsterCarnivalObtainedCPBody(cp uint16, total uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return carnivalpkt.NewMonsterCarnivalObtainedCP(cp, total).Encode(l, ctx)(options)
		}
	}
}
