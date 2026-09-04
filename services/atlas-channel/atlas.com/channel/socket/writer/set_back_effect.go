package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SetBackEffectBody(effect byte, fieldId uint32, pageId byte, duration uint32) packet.Encode {
	return fieldcb.NewSetBackEffect(effect, fieldId, pageId, duration).Encode
}
