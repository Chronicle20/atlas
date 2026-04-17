package shop

import (
	"atlas-merchant/data/portal"
	"context"
	"errors"
	"math"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

var (
	ErrNotFreemarketRoom = errors.New("not a free market room")
	ErrTooCloseToPortal  = errors.New("too close to a portal")
	ErrTooCloseToShop    = errors.New("too close to another shop")
	ErrPetItem           = errors.New("pets cannot be listed")
	ErrCashItem          = errors.New("cash items cannot be listed")
	ErrUntradeableItem   = errors.New("untradeable items cannot be listed")
)

const (
	portalProximityThreshold = 130
	shopProximityThreshold   = 100
)

var freeMarketRooms = map[uint32]bool{
	// Henesys Free Market <1> through <9>
	100000111: true, 100000112: true, 100000113: true,
	100000114: true, 100000115: true, 100000116: true,
	100000117: true, 100000118: true, 100000119: true,
	// Perion Free Market <1> through <9>
	102000101: true, 102000102: true, 102000103: true,
	102000104: true, 102000105: true, 102000106: true,
	102000107: true, 102000108: true, 102000109: true,
	// El Nath Free Market <1> through <5>
	211000111: true, 211000112: true, 211000113: true,
	211000114: true, 211000115: true,
	// Ludibrium Free Market <1> through <9>
	220000201: true, 220000202: true, 220000203: true,
	220000204: true, 220000205: true, 220000206: true,
	220000207: true, 220000208: true, 220000209: true,
	// Hidden Street Free Market <1> through <22>
	910000001: true, 910000002: true, 910000003: true,
	910000004: true, 910000005: true, 910000006: true,
	910000007: true, 910000008: true, 910000009: true,
	910000010: true, 910000011: true, 910000012: true,
	910000013: true, 910000014: true, 910000015: true,
	910000016: true, 910000017: true, 910000018: true,
	910000019: true, 910000020: true, 910000021: true,
	910000022: true,
}

func IsFreemarketRoom(mapId uint32) bool {
	return freeMarketRooms[mapId]
}

func IsNearPortal(l logrus.FieldLogger, ctx context.Context, mapId uint32, x int16, y int16) bool {
	portals, err := portal.GetByMapId(l, ctx)(mapId)()
	if err != nil {
		l.WithError(err).Warnf("Unable to fetch portal data for map [%d], skipping proximity check.", mapId)
		return false
	}
	for _, p := range portals {
		dist := manhattanDistance(x, y, p.X(), p.Y())
		if dist < portalProximityThreshold {
			return true
		}
	}
	return false
}

func IsNearExistingShop(mapId uint32, x int16, y int16, shopProvider model.Provider[[]Model]) bool {
	shops, err := shopProvider()
	if err != nil {
		return false
	}
	for _, s := range shops {
		if s.MapId() != mapId {
			continue
		}
		dist := manhattanDistance(x, y, s.X(), s.Y())
		if dist < shopProximityThreshold {
			return true
		}
	}
	return false
}

func IsListableItem(itemId uint32, flag uint16) error {
	classification := item.GetClassification(item.Id(itemId))
	if classification == item.ClassificationPet {
		return ErrPetItem
	}

	invType, ok := inventory.TypeFromItemId(item.Id(itemId))
	if ok && invType == inventory.TypeValueCash {
		return ErrCashItem
	}

	if asset.HasFlag(flag, asset.FlagUntradeable) {
		return ErrUntradeableItem
	}
	return nil
}

func manhattanDistance(x1, y1, x2, y2 int16) int {
	return int(math.Abs(float64(x1-x2))) + int(math.Abs(float64(y1-y2)))
}
