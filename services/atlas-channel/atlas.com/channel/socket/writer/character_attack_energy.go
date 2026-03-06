package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterAttackEnergy = "CharacterAttackEnergy"

func CharacterAttackEnergyBody(c character.Model, ai model.AttackInfo) packet.Encode {
	return WriteCommonAttackBody(c, ai)
}
