package serverbound

// SolomonItemUse is an AUDIT-ONLY codec. USE_SOLOMON_ITEM shares its wire body
// with USE_ITEM (Encode4 updateTime + Encode2 slot + Encode4 itemId), but its
// client send site is CWvsContext::SendExpUpItemUseRequest — a different
// function, gated on is_exp_up_item(nItemID) (itemId/10000 == 237) rather than
// the potion families. Collapsing it onto InventoryItemUse's packet id would
// pin every cell's evidence to the potion sender's decompile — a manufactured
// ✅. One wrapper per op = one packet id, one audit report and one evidence key
// per op, exactly as docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model
// ops" prescribes.
//
// Nothing calls this: atlas-channel's CharacterItemUseSolomonHandleFunc keeps
// decoding the shared ItemUse. Do not "simplify" it away (see task-229).
//
// packet-audit:fname CWvsContext::SendExpUpItemUseRequest
type SolomonItemUse struct {
	ItemUse
}

func NewSolomonItemUse() SolomonItemUse {
	return SolomonItemUse{ItemUse: NewItemUse(CharacterItemUseSolomonHandle)}
}
