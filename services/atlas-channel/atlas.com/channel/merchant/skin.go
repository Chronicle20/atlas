package merchant

// personalStorePermitBase is the item id of the plain (skin 0) store permit.
// Store permits are 5140000..514000N and the client selects the balloon sign
// skin from the permit's offset within the family.
const personalStorePermitBase = 5140000

// StoreSkinSpec maps a personal-store permit item id to the balloon's nSpec
// byte (CUser::OnMiniRoomBalloon 5th Decode1), which the client uses to pick
// the store-sign skin at WZ UI/ChatBalloon.img/miniroom/PSSkin/<nSpec>
// (CChatBalloon::MakeMiniRoomBalloon, GMS v95 @0x4a2d90, personal-shop case
// formats "PSSkin/%d" with nSpec). The mapping is nSpec = permitItemId - base:
// the WZ PSSkin canvases (0,1,2,3,4,6) are a 1:1 match with the store permits
// 5140000/1/2/3/4/6. Ids outside the permit family clamp to 0 (plain sign) so
// a stray value never indexes a missing skin node.
func StoreSkinSpec(permitItemId uint32) byte {
	if permitItemId < personalStorePermitBase || permitItemId > personalStorePermitBase+255 {
		return 0
	}
	return byte(permitItemId - personalStorePermitBase)
}
