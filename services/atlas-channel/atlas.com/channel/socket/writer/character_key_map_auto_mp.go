package writer

import (
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterKeyMapAutoMp = "CharacterKeyMapAutoMp"

func CharacterKeyMapAutoMpBody(action int32) packet.Encode {
	return charpkt.NewCharacterKeyMapAutoMp(action).Encode
}
