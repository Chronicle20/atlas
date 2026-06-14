package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SheepRanchClothesBody(characterId uint32, team byte) packet.Encode {
	return fieldcb.NewSheepRanchClothes(characterId, team).Encode
}
