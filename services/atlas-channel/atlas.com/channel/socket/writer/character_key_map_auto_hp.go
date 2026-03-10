package writer

import (
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterKeyMapAutoHp = "CharacterKeyMapAutoHp"

func CharacterKeyMapAutoHpBody(action int32) packet.Encode {
	return charpkt.NewCharacterKeyMapAutoHp(action).Encode
}
