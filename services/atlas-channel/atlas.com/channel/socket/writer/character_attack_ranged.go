package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterAttackRanged = "CharacterAttackRanged"

func CharacterAttackRangedBody(c character.Model, ai model.AttackInfo) packet.Encode {
	return WriteCommonAttackBody(c, ai)
}
