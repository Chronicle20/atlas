package writer

import (
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	CharacterSkillMacro = "CharacterSkillMacro"
)

func CharacterSkillMacroBody(m model.Macros) packet.Encode {
	return m.Encoder
}
