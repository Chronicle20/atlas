package writer

import (
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func ClaimSvrStatusChangedBody(connected bool) packet.Encode {
	return packet.Encode(reportcb.NewClaimSvrStatusChanged(connected).Encode)
}
