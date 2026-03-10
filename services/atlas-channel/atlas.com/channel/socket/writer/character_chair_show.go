package writer

import (
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterShowChair = "CharacterShowChair"

func CharacterShowChairBody(characterId uint32, chairId uint32) packet.Encode {
	return charpkt.NewCharacterChairShow(characterId, chairId).Encode
}
