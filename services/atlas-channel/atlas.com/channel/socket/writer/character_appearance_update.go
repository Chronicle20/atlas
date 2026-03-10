package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"

	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterAppearanceUpdate = "CharacterAppearanceUpdate"

func CharacterAppearanceUpdateBody(c character.Model) packet.Encode {
	ava := model.NewFromCharacter(c, false)
	return charpkt.NewCharacterAppearanceUpdate(c.Id(), ava).Encode
}
