package writer

import (
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterHint = "CharacterHint"

func CharacterHintBody(hint string, width uint16, height uint16, atPoint bool, x int32, y int32) packet.Encode {
	return charpkt.NewCharacterHint(hint, width, height, atPoint, x, y).Encode
}
