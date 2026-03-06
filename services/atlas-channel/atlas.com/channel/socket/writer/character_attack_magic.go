package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterAttackMagic = "CharacterAttackMagic"

func CharacterAttackMagicBody(c character.Model, ai model.AttackInfo) packet.Encode {
	return WriteCommonAttackBody(c, ai)
}
