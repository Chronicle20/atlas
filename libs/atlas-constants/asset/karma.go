package asset

import "github.com/Chronicle20/atlas/libs/atlas-constants/item"

// KarmaFlagFor returns the karma-mark bit for an asset's slot class, plus
// whether karma applies to that class at all.
//
// The class is derived from the template id exactly as the client derives it:
// CItemInfo::GetAppliableKarmaType (gms_v95 @0x5C09F0) branches on
// nItemID / 1000000 == 1 to choose EQUIPITEM over BUNDLEITEM, and the two
// GW_ItemSlot subclasses read different bits out of nAttribute:
//
//	GW_ItemSlotEquip::IsPossibleTradingItem  gms_v83 @0x4E956E / gms_v95 @0x4F6130 -> nAttribute & 0x10
//	GW_ItemSlotBundle::IsPossibleTradingItem gms_v83 @0x4E9B6A / gms_v95 @0x4F67A0 -> nAttribute & 0x02
//
// Pets are DELIBERATELY refused rather than resolved. The client's
// GW_ItemSlotPet::IsPossibleTradingItem (gms_v83 @0x4EA01E) reads 0x01, which
// it can afford because GW_ItemSlotPet::IsProtectedItem (@0x4EA012) hard-returns
// 0 — the two meanings never coexist on one client object. Atlas has no such
// guarantee: FlagLock is written against the same shared `flag` column by the
// Sealing Lock arm, so a pet karma mark would read back as a lock. Returning
// (0, false) forces every caller to handle the case rather than silently
// writing 0x01. See design.md OQ-5.
func KarmaFlagFor(templateId uint32) (Flag, bool) {
	if item.GetClassification(item.Id(templateId)) == item.ClassificationPet {
		return 0, false
	}
	if templateId/1000000 == 1 {
		return FlagKarmaEquip, true
	}
	return FlagKarmaUse, true
}

// KarmaEligible reports whether a Scissors of Karma carrying scissorsKarma
// (WZ info/karma on the scissors, 0 when absent) may be applied to a target
// carrying targetKarma (WZ info/tradeAvailable on the target, 0 when absent).
//
// One predicate covers both client models (design §3.2-3.3):
//
//   - gms_v83 asks "is tradeAvailable non-zero?" (CItemInfo::IsAppliableKarmaItem
//     @0x5D4E8F). Its scissors carry no `karma` node, so scissorsKarma is 0, the
//     second clause is vacuous, and this reduces to exactly that test.
//   - gms_v87 (CUIKarmaDlg::PutItem @0x895261) and gms_v95 (@0x7D7BA0) ask
//     "does GetAppliableKarmaType(target) equal m_nKarmaType?". Their scissors
//     carry a `karma` node, so this is that equality plus one extra condition.
//
// The extra condition — targetKarma != 0 — closes a real client hole: a v95-era
// tenant whose WZ omits `karma` on the scissors makes the client compare
// 0 != tradeAvailable and thereby accept every ordinary item. The server must
// not. Being a strict subset of both client rules, the worst case on any
// un-decompiled version is a logged server refusal where the client would have
// allowed.
func KarmaEligible(scissorsKarma int32, targetKarma int32) bool {
	if targetKarma == 0 {
		return false
	}
	return scissorsKarma == 0 || targetKarma == scissorsKarma
}
