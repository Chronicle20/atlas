// Package mapdata is the atlas-data map REST client atlas-trades validates
// through. It reads only the `fieldLimit` bitmask, which backs FR-4.6's
// "trade is refused on this map" rule.
package mapdata

import _map "github.com/Chronicle20/atlas/libs/atlas-constants/map"

// Model is the minimal map view the trade validation ladder needs: the WZ
// `fieldLimit` bitmask (see TradeDisallowed for the bit that matters).
type Model struct {
	id         _map.Id
	fieldLimit uint32
}

func (m Model) Id() _map.Id { return m.id }

func (m Model) FieldLimit() uint32 { return m.fieldLimit }
