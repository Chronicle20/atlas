package writer

import (
	"context"

	carnivalpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MonsterCarnivalSummonBody encodes the clientbound MONSTER_CARNIVAL_SUMMON packet
// (CField_MonsterCarnival::OnRequestResult, SUMMON mode), confirming a summon
// request. No emitter wires this writer yet; it is an intentional seam.
func MonsterCarnivalSummonBody(tab byte, idx byte, name string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return carnivalpkt.NewMonsterCarnivalSummon(tab, idx, name).Encode(l, ctx)(options)
		}
	}
}
