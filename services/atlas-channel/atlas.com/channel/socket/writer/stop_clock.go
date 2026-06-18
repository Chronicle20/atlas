package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func StopClockBody() packet.Encode {
	return fieldcb.NewStopClock().Encode
}
