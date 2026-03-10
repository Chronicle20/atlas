package writer

import (
	"time"

	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	CharacterSkillCooldown = "CharacterSkillCooldown"
)

func CharacterSkillCooldownBody(skillId uint32, cooldownExpiresAt time.Time) packet.Encode {
	var cd uint16
	if !cooldownExpiresAt.IsZero() {
		cd = uint16(cooldownExpiresAt.Sub(time.Now()).Seconds())
	}
	return charpkt.NewCharacterSkillCooldown(skillId, cd).Encode
}
