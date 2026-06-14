package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func PyramidGaugeBody(gauge uint32) packet.Encode {
	return fieldcb.NewPyramidGauge(gauge).Encode
}
