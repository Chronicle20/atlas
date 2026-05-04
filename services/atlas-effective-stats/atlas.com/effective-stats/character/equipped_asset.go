package character

import "atlas-effective-stats/stat"

// EquippedAsset is the per-asset snapshot held on the character Model.
// It is the source of truth for equipment bonuses; m.bonuses[] holds only
// buff:* and passive:* entries.
type EquippedAsset struct {
	assetId    uint32
	templateId uint32
	bonuses    []stat.Bonus
}

// NewEquippedAsset takes a defensive copy of bonuses so callers can mutate
// their slice without affecting the snapshot.
func NewEquippedAsset(assetId, templateId uint32, bonuses []stat.Bonus) EquippedAsset {
	owned := make([]stat.Bonus, len(bonuses))
	copy(owned, bonuses)
	return EquippedAsset{
		assetId:    assetId,
		templateId: templateId,
		bonuses:    owned,
	}
}

func (a EquippedAsset) AssetId() uint32    { return a.assetId }
func (a EquippedAsset) TemplateId() uint32 { return a.templateId }

// Bonuses returns a defensive copy of the snapshot's flat bonuses.
func (a EquippedAsset) Bonuses() []stat.Bonus {
	out := make([]stat.Bonus, len(a.bonuses))
	copy(out, a.bonuses)
	return out
}
