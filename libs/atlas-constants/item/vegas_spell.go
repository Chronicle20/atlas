package item

// Vega's Spell cash consumables (Item.wz/Cash/0561.img.xml). Using one
// consumes the vega item together with an upgrade scroll whose natural
// success rate matches the variant exactly, applying the scroll at a boosted
// rate. The rate pairing itself (10→30, 60→90) is server policy and lives in
// atlas-consumables — these are wire/domain identities only.
const (
	// VegasSpell10 boosts a 10% scroll to 30%.
	VegasSpell10 = Id(5610000)

	// VegasSpell60 boosts a 60% scroll to 90%.
	VegasSpell60 = Id(5610001)

	// ClassificationVegasSpell is the cash-compartment classification
	// (item id / 10000) for Vega's Spell items.
	ClassificationVegasSpell = Classification(561)
)

// IsVegasSpell returns true if the item is a Vega's Spell variant.
func IsVegasSpell(id Id) bool {
	return Is(id, VegasSpell10, VegasSpell60)
}
