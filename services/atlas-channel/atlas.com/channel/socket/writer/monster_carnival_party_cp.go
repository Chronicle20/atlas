package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	carnivalpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MonsterCarnivalPartyCPBody encodes the clientbound MONSTER_CARNIVAL_PARTY_CP
// packet (CField_MonsterCarnival::OnTeamCP), a team CP scoreboard update.
// No emitter wires this writer yet; it is an intentional seam.
func MonsterCarnivalPartyCPBody(team byte, cp uint16, total uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return carnivalpkt.NewMonsterCarnivalPartyCP(team, cp, total).Encode(l, ctx)(options)
		}
	}
}
