package writer

import (
	"context"

	carnivalpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MonsterCarnivalLeaveBody encodes the clientbound MONSTER_CARNIVAL_LEAVE packet
// (CField_MonsterCarnival::OnShowMemberOutMsg), a participant-quit announcement.
// No emitter wires this writer yet; it is an intentional seam.
func MonsterCarnivalLeaveBody(leader byte, team byte, name string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return carnivalpkt.NewMonsterCarnivalLeave(leader, team, name).Encode(l, ctx)(options)
		}
	}
}
