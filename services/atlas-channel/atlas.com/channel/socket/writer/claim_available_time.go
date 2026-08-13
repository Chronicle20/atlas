package writer

import (
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func ClaimAvailableTimeBody(openHour byte, closeHour byte) packet.Encode {
	return packet.Encode(reportcb.NewClaimAvailableTime(openHour, closeHour).Encode)
}
