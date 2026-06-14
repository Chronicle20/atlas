package writer

import (
	"context"

	carnivalpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MonsterCarnivalStartBody encodes the clientbound MONSTER_CARNIVAL_START packet
// (CField_MonsterCarnival::OnEnter), the initial carnival scoreboard state.
// No emitter wires this writer yet; it is an intentional seam.
func MonsterCarnivalStartBody(team byte, personalCp uint16, personalTotal uint16, myTeamCp uint16, myTeamTotal uint16, enemyTeamCp uint16, enemyTeamTotal uint16, spelled []byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return carnivalpkt.NewMonsterCarnivalStart(team, personalCp, personalTotal, myTeamCp, myTeamTotal, enemyTeamCp, enemyTeamTotal, spelled).Encode(l, ctx)(options)
		}
	}
}
