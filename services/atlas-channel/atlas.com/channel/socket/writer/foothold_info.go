package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func FootholdInfoBody(entries []fieldcb.FootholdEntry) packet.Encode {
	return fieldcb.NewFootholdInfo(entries).Encode
}
