package writer

import (
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterItemUpgrade = "CharacterItemUpgrade"

func CharacterItemUpgradeBody(characterId uint32, success bool, cursed bool, legendarySpirit bool, whiteScroll bool) packet.Encode {
	return charpkt.NewItemUpgrade(characterId, success, cursed, legendarySpirit, whiteScroll).Encode
}
