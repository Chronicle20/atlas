package serverbound

// ReturnScrollItemUse is an AUDIT-ONLY codec. USE_RETURN_SCROLL shares its wire
// body with USE_ITEM and USE_SUMMON_BAG (Encode4 updateTime + Encode2 slot +
// Encode4 itemId) but has its own client send site — on v61/v72/v79 that sender
// reads the Return Scroll WZ props and is distinct from both the potion sender
// and the teleport-rock sender (see the corrections recorded in
// docs/packets/registry/gms_v72.yaml and gms_v79.yaml). One wrapper per op = one
// packet id, one audit report and one evidence key per op, per
// docs/packets/audits/VERIFYING_A_PACKET.md "Shared-model ops".
//
// Nothing calls this: atlas-channel's CharacterItemUseTownScrollHandleFunc keeps
// decoding the shared ItemUse. Do not "simplify" it away (task-229).
//
// packet-audit:fname CWvsContext::SendPortalScrollUseRequest
type ReturnScrollItemUse struct {
	ItemUse
}

func NewReturnScrollItemUse() ReturnScrollItemUse {
	return ReturnScrollItemUse{ItemUse: NewItemUse(CharacterItemUseTownScrollHandle)}
}
