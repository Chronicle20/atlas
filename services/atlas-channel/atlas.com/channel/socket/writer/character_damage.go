package writer

import (
	"atlas-channel/character"

	charpkt "github.com/Chronicle20/atlas-packet/character"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterDamage = "CharacterDamage"

func CharacterDamageBody(c character.Model, di packetmodel.DamageTakenInfo) packet.Encode {
	return charpkt.NewCharacterDamageW(c.Id(), di.AttackIdx(), di.Damage(), di.MonsterTemplateId(), di.Left()).Encode
}
