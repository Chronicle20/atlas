package merchant

import (
	merchantconst "github.com/Chronicle20/atlas/libs/atlas-constants/merchant"
	interactionpkt "github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
)

// HiredMerchantShopType is the shared atlas-constants ShopType for a hired
// merchant, exposed as byte to match the wire model. Personal stores render
// as a box on the owner's avatar; hired merchants render as a standalone
// CEmployeePool NPC.
const HiredMerchantShopType = byte(merchantconst.ShopTypeHiredMerchant)

// ToEmployeeSpawn projects a hired-merchant shop into its CEmployeePool field
// spawn packet (SPAWN_HIRED_MERCHANT). The employeeId and the balloon serial are
// the owner characterId: a character runs at most one merchant, and CEmployeePool
// is keyed independently of the user pool, so it is a stable, collision-free key.
// templateId is the store-permit item id (503xxxx) — the client resolves the
// sprite from Item/Cash/0503.img's info/employee node keyed by that id. foothold 0
// is client-guarded (CEmployee::Init's GetFoothold guards id 0); the on-ground
// placement x/y drive the sprite position. The balloon carries the store type
// (MerchantShop=5) and title so other players see and can enter the shop.
func ToEmployeeSpawn(m Model, ownerName string) merchantcb.EmployeeSpawn {
	balloon := merchantcb.NewBalloon(
		byte(interactionpkt.MerchantShopMiniRoomType),
		m.CharacterId(),
		m.Title(),
		byte(len(m.Visitors())),
		4,
		0,
	)
	return merchantcb.NewEmployeeSpawn(m.CharacterId(), m.PermitItemId(), m.X(), m.Y(), 0, ownerName, balloon)
}

// ToEmployeeUpdate projects the current balloon state of a hired-merchant shop
// into the UPDATE packet (CEmployeePool::OnEmployeeMiniRoomBalloon), refreshing the
// field balloon — e.g. the visitor count — for players already in the map. It
// reuses the same balloon block as the spawn, keyed by the owner characterId.
func ToEmployeeUpdate(m Model) merchantcb.EmployeeUpdate {
	balloon := merchantcb.NewBalloon(
		byte(interactionpkt.MerchantShopMiniRoomType),
		m.CharacterId(),
		m.Title(),
		byte(len(m.Visitors())),
		4,
		0,
	)
	return merchantcb.NewEmployeeUpdate(m.CharacterId(), balloon)
}
