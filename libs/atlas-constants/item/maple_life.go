package item

// The two Maple Life cash-shop character-creation coupon ids. Both carry
// ClassificationCharacterCreation (543), but String.wz Cash.img is explicit
// that they are NOT the same product (task-246 bug-b-type-must-add-a-slot.md):
//
//   - MapleLifeATypeId (5431000): "*Warning: If you do not have an empty
//     Character slot, it cannot be used." -- rejected at the current slot
//     count, same as any other character-creation flow.
//   - MapleLifeBTypeId (5432000): "*Warning: If all 12 character slots are
//     full, it cannot be used." -- this package bundles an Extra Character
//     Slot coupon, so using it ADDS a slot (up to the 12 cap) rather than
//     merely consuming one.
//
// (5430000, the standalone "Extra Character Slot Coupon", is a third,
// separate id that never reaches the Maple Life creation handler -- see
// character_cash_item_use.go's 543 branch and the bug file's derivation
// note. It has no constant here because nothing in this codebase currently
// keys logic off it.)
const (
	MapleLifeATypeId = Id(5431000)
	MapleLifeBTypeId = Id(5432000)
)
