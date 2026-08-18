package serverbound

// SummonBagItemUse is an AUDIT-ONLY codec. USE_SUMMON_BAG shares its wire body
// with USE_ITEM and USE_RETURN_SCROLL (Encode4 updateTime + Encode2 slot +
// Encode4 itemId), but each op has a DIFFERENT client send site. Collapsing all
// three onto InventoryItemUse's packet id would pin every cell's evidence to the
// potion sender's decompile — a manufactured ✅. One wrapper per op = one packet
// id, one audit report and one evidence key per op, exactly as
// docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model ops" prescribes.
//
// Nothing calls this: atlas-channel's CharacterItemUseSummonBagHandleFunc keeps
// decoding the shared ItemUse. Do not "simplify" it away (task-229).
//
// packet-audit:fname CWvsContext::SendMobSummonItemUseRequest
type SummonBagItemUse struct {
	ItemUse
}

func NewSummonBagItemUse() SummonBagItemUse {
	return SummonBagItemUse{ItemUse: NewItemUse(CharacterItemUseSummonBagHandle)}
}
