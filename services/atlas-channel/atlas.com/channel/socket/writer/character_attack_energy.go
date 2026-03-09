package writer

import (
	"atlas-channel/character"

	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterAttackEnergy = "CharacterAttackEnergy"

func CharacterAttackEnergyBody(c character.Model, ai packetmodel.AttackInfo) packet.Encode {
	return WriteCommonAttackBody(c, ai)
}
