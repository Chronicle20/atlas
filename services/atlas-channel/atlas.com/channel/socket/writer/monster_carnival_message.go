package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	carnivalpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MonsterCarnivalMessageBody encodes the clientbound MONSTER_CARNIVAL_MESSAGE packet
// (CField_MonsterCarnival::OnRequestResult, MESSAGE mode), a status-line selector.
// No emitter wires this writer yet; it is an intentional seam.
func MonsterCarnivalMessageBody(message byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return carnivalpkt.NewMonsterCarnivalMessage(message).Encode(l, ctx)(options)
		}
	}
}
