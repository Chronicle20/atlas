package shop

import (
	"atlas-merchant/data/portal"
	asset2 "atlas-merchant/kafka/message/asset"
	"context"
	"errors"
	"math"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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
	// portalProximityThreshold is the euclidean radius within which a teleport
	// portal blocks store placement (within 120px of the closest teleport
	// portal the client-era rule rejects placement).
	portalProximityThreshold = 120
	shopProximityThreshold   = 100

	// teleportPortalType and portalTargetNone identify store-blocking portals
	// in map data: only TELEPORT portals (WZ type 1)
	// with a real target map block store placement. Spawn points ("sp",
	// type 0 — where players land entering the Free Market) and dead-end portals
	// are ignored, so a store can be opened where the player stands.
	teleportPortalType = 1
	portalTargetNone   = 999999999
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
	return nearBlockingPortal(x, y, portals)
}

// nearBlockingPortal reports whether (x,y) is within portalProximityThreshold of
// a store-blocking portal. Only teleport portals with a real target block
// placement — spawn points, where players stand on entering the
// Free Market, are ignored, so a store can be opened there.
func nearBlockingPortal(x int16, y int16, portals []portal.Model) bool {
	limitSq := portalProximityThreshold * portalProximityThreshold
	for _, p := range portals {
		if p.Type() != teleportPortalType || p.TargetMapId() == portalTargetNone {
			continue
		}
		if squaredDistance(x, y, p.X(), p.Y()) < limitSq {
			return true
		}
	}
	return false
}

func squaredDistance(x1, y1, x2, y2 int16) int {
	dx := int(x1) - int(x2)
	dy := int(y1) - int(y2)
	return dx*dx + dy*dy
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

	// A karma mark (Scissors of Karma, task-223) buys exactly one transfer, and a
	// hired-merchant SALE is one — so a marked item lists, and the mark is
	// consumed when the buyer's asset is built (see the buy path in
	// processor.go). Listing-only semantics would let a player launder one mark
	// into unlimited transfers by re-listing. ErrPetItem and ErrCashItem above
	// are untouched: they are not tradeability rules.
	//
	// The bit is slot-class dependent — 0x02 on an EQUIP is FlagSpikes, so a
	// spiked untradeable equip must still be refused. KarmaFlagFor is the only
	// thing that may pick it.
	karmaMarked := false
	if f, ok := asset.KarmaFlagFor(itemId); ok {
		karmaMarked = asset.HasFlag(flag, f)
	}
	if !karmaMarked && asset.HasFlag(flag, asset.FlagUntradeable) {
		return ErrUntradeableItem
	}
	return nil
}

// clearKarmaFromAssetData masks the karma mark off the snapshot used to build
// the BUYER's asset. Same rationale as the trade path (atlas-saga-orchestrator's
// clearKarmaFromSnapshot): the clear and the transfer are the same write, so
// there is no window in which the delivered item still carries a free trade.
//
// Applied ONLY where ownership changes hands. The three "return the item to its
// owner" paths — shop closure, listing removal, Frederick retrieval — pass the
// snapshot through untouched, exactly as a cancelled trade does.
func clearKarmaFromAssetData(itemId uint32, ad asset2.AssetData) asset2.AssetData {
	f, ok := asset.KarmaFlagFor(itemId)
	if !ok {
		return ad
	}
	ad.Flag = asset.ClearFlag(ad.Flag, f)
	return ad
}

func manhattanDistance(x1, y1, x2, y2 int16) int {
	return int(math.Abs(float64(x1-x2))) + int(math.Abs(float64(y1-y2)))
}
