package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SnowballMessageBody(team byte, message byte) packet.Encode {
	return fieldcb.NewSnowballMessage(team, message).Encode
}
