package writer

import (
	"time"

	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	CharacterSkillChange = "CharacterSkillChange"
)

func CharacterSkillChangeBody(exclRequestSent bool, skillId uint32, level byte, masterLevel byte, expiration time.Time, sn bool) packet.Encode {
	return charpkt.NewCharacterSkillChange(exclRequestSent, skillId, level, masterLevel, expiration, sn).Encode
}
