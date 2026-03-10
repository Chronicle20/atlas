package writer

import (
	"github.com/Chronicle20/atlas-socket/packet"

	invpkt "github.com/Chronicle20/atlas-packet/inventory"
)

const CompartmentMerge = "CompartmentMerge"

func CompartmentMergeBody(inventoryType byte) packet.Encode {
	return invpkt.NewCompartmentMergeW(inventoryType).Encode
}
