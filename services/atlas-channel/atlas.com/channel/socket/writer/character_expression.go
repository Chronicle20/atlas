package writer

import (
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterExpression = "CharacterExpression"

func CharacterExpressionBody(characterId uint32, expression uint32) packet.Encode {
	return charpkt.NewCharacterExpressionW(characterId, expression).Encode
}
