package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func FieldObstacleOnOffListBody(obstacles []fieldcb.ObstacleState) packet.Encode {
	return fieldcb.NewFieldObstacleOnOffList(obstacles).Encode
}
