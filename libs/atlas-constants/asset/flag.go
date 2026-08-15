package asset

type Flag uint16

// The client reads these bits out of GW_ItemSlotBase::nAttribute, and TWO of
// them are SLOT-CLASS DEPENDENT — the same value means different things on an
// equip, a bundle and a pet:
//
//	bit    equip                      bundle                     pet
//	0x01   lock (IsProtectedItem)     lock                       karma (IsProtectedItem hard-returns 0)
//	0x02   spikes                     KARMA MARK                 --
//	0x10   KARMA MARK                 --                         --
//
// gms_v83: GW_ItemSlotEquip::IsProtectedItem @0x4E9506, ::IsPossibleTradingItem
// @0x4E956E; GW_ItemSlotBundle::IsProtectedItem @0x4E9B4F,
// ::IsPossibleTradingItem @0x4E9B6A; GW_ItemSlotPet::IsProtectedItem @0x4EA012,
// ::IsPossibleTradingItem @0x4EA01E.
// gms_v95: equip @0x4F60B0 / @0x4F6130; bundle @0x4F6780 / @0x4F67A0.
//
// The names below read BACKWARDS from that table and are load-bearing in seven
// services, so they are documented rather than renamed: FlagKarmaUse (0x02) is
// the BUNDLE karma bit and FlagKarmaEquip (0x10) is the EQUIP karma bit. Never
// pick one by hand — call KarmaFlagFor (karma.go), which resolves the bit from
// the template id's slot class and refuses pets.
//
// The values are the client's and MUST NOT change.
const (
	FlagLock             Flag = 0x01
	FlagSpikes           Flag = 0x02
	FlagKarmaUse         Flag = 0x02
	FlagCold             Flag = 0x04
	FlagUntradeable      Flag = 0x08
	FlagKarmaEquip       Flag = 0x10
	FlagSandbox          Flag = 0x40
	FlagPetCome          Flag = 0x80
	FlagAccountSharing   Flag = 0x100
	FlagMergeUntradeable Flag = 0x200
)

func HasFlag(flags uint16, f Flag) bool {
	return flags&uint16(f) != 0
}

func SetFlag(flags uint16, f Flag) uint16 {
	return flags | uint16(f)
}

func ClearFlag(flags uint16, f Flag) uint16 {
	return flags &^ uint16(f)
}
