package writer

import (
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterSitResult = "CharacterSitResult"

func CharacterSitBody(chairId uint16) packet.Encode {
	return charpkt.NewCharacterSit(chairId).Encode
}

func CharacterCancelSitBody() packet.Encode {
	return charpkt.NewCharacterCancelSit().Encode
}
