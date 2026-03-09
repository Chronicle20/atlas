package writer

import (
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	CharacterSkillMacro = "CharacterSkillMacro"
)

func CharacterSkillMacroBody(m packetmodel.Macros) packet.Encode {
	return m.Encode
}
