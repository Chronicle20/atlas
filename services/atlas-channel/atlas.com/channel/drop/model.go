package drop

import (
	"time"
)

type Model struct {
	id           uint32
	itemId       uint32
	equipmentId  uint32
	quantity     uint32
	meso         uint32
	dropType     byte
	x            int16
	y            int16
	ownerId      uint32
	ownerPartyId uint32
	dropTime     time.Time
	dropperId    uint32
	dropperX     int16
	dropperY     int16
	playerDrop   bool
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) Meso() uint32 {
	return m.meso
}

func (m Model) Type() byte {
	return m.dropType
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}

func (m Model) OwnerPartyId() uint32 {
	return m.ownerPartyId
}

func (m Model) DropTime() time.Time {
	return m.dropTime
}

func (m Model) DropperId() uint32 {
	return m.dropperId
}

func (m Model) DropperX() int16 {
	return m.dropperX
}

func (m Model) DropperY() int16 {
	return m.dropperY
}

func (m Model) PlayerDrop() bool {
	return m.playerDrop
}

func (m Model) CharacterDrop() bool {
	return m.playerDrop
}

func (m Model) EquipmentId() uint32 {
	return m.equipmentId
}

func (m Model) Owner() uint32 {
	if m.ownerPartyId != 0 {
		return m.ownerPartyId
	}
	return m.ownerId
}

// OwnType returns the ownership-window discriminant that must accompany
// Owner() in DropSpawn/DropEnterField. Client evidence (GMS v83.1
// MapleStory_dump.exe.i64, session 754107bf): CDropPool::TryPickUpDrop
// @0x50463c gates SendDropPickUpRequest on
//
//	ownType == 0 -> owner (dwOwner, +0x24) must equal the local character id
//	ownType == 1 -> owner must equal the local party id
//	ownType >= 2 -> no owner check (FFA / explosive drop types)
//
// Owner() substitutes the party id for the character id whenever the drop is
// party-owned, so OwnType must switch to 1 in lockstep or every client's
// comparison fails and the 15s ownership window never lifts.
func (m Model) OwnType() byte {
	if m.dropType >= 2 {
		return m.dropType
	}
	if m.ownerPartyId != 0 {
		return 1
	}
	return 0
}
