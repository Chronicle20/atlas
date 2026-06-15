package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// StalkResultBody builds the IDA_0X09C / OnStalkResult (CField::OnStalkResult)
// clientbound packet — the minimap stalkee-list update. The codec + route exist so
// the stalk feature can be switched on later without another packet-plumbing pass;
// the uncalled writer is a documented seam (IMPLEMENTING_A_PACKET D2), not dead code.
func StalkResultBody(count uint32, charId uint32, flag byte, name string, x uint32, y uint32) packet.Encode {
	return fieldcb.NewStalkResult(count, charId, flag, name, x, y).Encode
}
