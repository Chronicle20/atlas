package writer

import (
	"context"

	carnivalpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MonsterCarnivalDiedBody encodes the clientbound MONSTER_CARNIVAL_DIED packet
// (CField_MonsterCarnival::OnProcessForDeath), a participant-defeated announcement.
// No emitter wires this writer yet; it is an intentional seam.
func MonsterCarnivalDiedBody(team byte, name string, lostCp byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return carnivalpkt.NewMonsterCarnivalDied(team, name, lostCp).Encode(l, ctx)(options)
		}
	}
}
