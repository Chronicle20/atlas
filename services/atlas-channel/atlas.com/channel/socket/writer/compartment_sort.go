package writer

import (
	"github.com/Chronicle20/atlas-socket/packet"

	invpkt "github.com/Chronicle20/atlas-packet/inventory"
)

const CompartmentSort = "CompartmentSort"

func CompartmentSortBody(inventoryType byte) packet.Encode {
	return invpkt.NewCompartmentSortW(inventoryType).Encode
}
