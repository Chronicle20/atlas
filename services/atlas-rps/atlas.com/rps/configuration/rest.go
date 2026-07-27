package configuration

import (
	"atlas-rps/game"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// RungRestModel is the JSON:API attribute shape of a single rung embedded in
// the rps-rewards `ladder` array. It is a plain nested JSON attribute (not a
// JSON:API relationship), so it decodes directly via encoding/json into the
// concrete field types below - no float64 intermediate is involved.
type RungRestModel struct {
	Rung     int     `json:"rung"`
	ItemId   item.Id `json:"itemId"`
	Quantity uint32  `json:"quantity"`
	Meso     uint32  `json:"meso"`
}

// RpsRewardRestModel is the JSON:API resource for the rps-rewards
// configuration served by atlas-tenants.
type RpsRewardRestModel struct {
	Id              string          `json:"-"`
	EntryCostMeso   uint32          `json:"entryCostMeso"`
	ConsolationMeso uint32          `json:"consolationMeso"`
	Ladder          []RungRestModel `json:"ladder"`
}

// GetID returns the resource ID.
func (r RpsRewardRestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID.
func (r *RpsRewardRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// GetName returns the resource name.
func (r RpsRewardRestModel) GetName() string {
	return "rps-rewards"
}

// Extract converts a RpsRewardRestModel to a game.Ladder. Rungs are copied in
// the order supplied by the configuration, skipping any rung number already
// seen so the resulting slice is dense and deduplicated (game.Ladder.PrizeAt
// matches by Rung field value and takes the first match, so a duplicate would
// silently shadow a later, correct entry).
func Extract(r RpsRewardRestModel) (game.Ladder, error) {
	seen := make(map[int]bool, len(r.Ladder))
	rungs := make([]game.Rung, 0, len(r.Ladder))
	for _, rr := range r.Ladder {
		if seen[rr.Rung] {
			continue
		}
		seen[rr.Rung] = true
		rungs = append(rungs, game.Rung{
			Rung:     rr.Rung,
			ItemId:   rr.ItemId,
			Quantity: rr.Quantity,
			Meso:     rr.Meso,
		})
	}
	return game.Ladder{
		EntryCostMeso:   r.EntryCostMeso,
		ConsolationMeso: r.ConsolationMeso,
		Rungs:           rungs,
	}, nil
}
