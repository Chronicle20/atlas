package teleportrock

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MAP_TRANSFER_RESULT mode keys, resolved per-version from the tenant
// template's "operations" table (design §7). Never hard-code the byte values.
const (
	MapTransferModeDeleteList        = "DELETE_LIST"
	MapTransferModeRegisterList      = "REGISTER_LIST"
	MapTransferModeCannotGo          = "CANNOT_GO"
	MapTransferModeUnableToLocate    = "UNABLE_TO_LOCATE"
	MapTransferModeUnableToLocate2   = "UNABLE_TO_LOCATE_2"
	MapTransferModeCannotGoContinent = "CANNOT_GO_CONTINENT"
	MapTransferModeCurrentMap        = "CURRENT_MAP"
	MapTransferModeMapNotAvailable   = "MAP_NOT_AVAILABLE"
	MapTransferModeMapleIslandLevel7 = "MAPLE_ISLAND_LEVEL7"
)

// MapTransferResultListBody emits the list-refresh form (REGISTER_LIST /
// DELETE_LIST): the full post-mutation list for the affected list, padded to
// 5/10 with EmptyMapId. The client only updates its UI from this packet.
func MapTransferResultListBody(key string, vip bool, maps []_map.Id) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", key, func(mode byte) packet.Encoder {
		return clientbound.NewMapTransferList(mode, vip, maps)
	})
}

// MapTransferResultErrorBody emits an error mode (CANNOT_GO, UNABLE_TO_LOCATE,
// CANNOT_GO_CONTINENT, CURRENT_MAP, MAP_NOT_AVAILABLE, ...).
func MapTransferResultErrorBody(key string, vip bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", key, func(mode byte) packet.Encoder {
		return clientbound.NewMapTransferError(mode, vip)
	})
}
