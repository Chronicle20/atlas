package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const DropDestroyWriter = "DropDestroy"

type DropDestroyType byte

const (
	DropDestroyTypeExpire    DropDestroyType = 0
	DropDestroyTypeNone      DropDestroyType = 1
	DropDestroyTypePickUp    DropDestroyType = 2
	DropDestroyTypeUnk1      DropDestroyType = 3
	DropDestroyTypeExplode   DropDestroyType = 4
	DropDestroyTypePetPickUp DropDestroyType = 5
)

// Wire shape verified against v95 IDA CDropPool::OnDropLeaveField@0x511e20:
//   byte(destroyType) + int(dropId)
//   if destroyType in {2, 3}: int(pickupCharId)
//   if destroyType == 4:      int16(tLeaveDelay)
//   if destroyType == 5:      int(pickupCharId) + int(petPickupExtra)
//
// The legacy `petSlot int8` field was a wire bug — v95 reads int4 inside
// case 5 (pet pickup), not a byte. NewDropDestroy preserves the legacy
// constructor signature for backwards compatibility and maps petSlot to
// petPickupExtra for type 5 paths. For type 4 (explode), legacy callers
// pass characterId=0 / petSlot=-1; the corrected wire emits int16(0) for
// the explode delay.
type Destroy struct {
	dropId           uint32
	destroyType      DropDestroyType
	characterId      uint32
	explodeDelay     int16
	petPickupExtra   uint32
}

// NewDropDestroy preserves the pre-task-065 signature. petSlot is only
// meaningful for destroyType == 5 (pet pickup), where it widens to int4
// to match v95's wire shape. For destroyType == 4 (explode) the legacy
// characterId/petSlot params are ignored and the wire emits int16(0)
// for the explode delay — callers that need a non-zero delay should
// use NewDropDestroyExplode below.
func NewDropDestroy(dropId uint32, destroyType DropDestroyType, characterId uint32, petSlot int8) Destroy {
	d := Destroy{dropId: dropId, destroyType: destroyType, characterId: characterId}
	if destroyType == DropDestroyTypePetPickUp && petSlot >= 0 {
		d.petPickupExtra = uint32(petSlot)
	}
	return d
}

// NewDropDestroyExplode emits the destroyType=4 wire shape with the
// trailing tLeaveDelay int16 the v95 client reads to time the explode
// animation. Wire: byte(4) + int(dropId) + int16(delay) = 7 bytes.
func NewDropDestroyExplode(dropId uint32, leaveDelay int16) Destroy {
	return Destroy{
		dropId:       dropId,
		destroyType:  DropDestroyTypeExplode,
		explodeDelay: leaveDelay,
	}
}

func (m Destroy) DropId() uint32                { return m.dropId }
func (m Destroy) DestroyType() DropDestroyType  { return m.destroyType }
func (m Destroy) CharacterId() uint32           { return m.characterId }
func (m Destroy) ExplodeDelay() int16           { return m.explodeDelay }
func (m Destroy) PetPickupExtra() uint32        { return m.petPickupExtra }
func (m Destroy) Operation() string             { return DropDestroyWriter }
func (m Destroy) String() string {
	return fmt.Sprintf("dropId [%d], destroyType [%d]", m.dropId, m.destroyType)
}

func (m Destroy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.destroyType))
		w.WriteInt(m.dropId)
		switch m.destroyType {
		case DropDestroyTypePickUp, DropDestroyTypeUnk1:
			w.WriteInt(m.characterId)
		case DropDestroyTypeExplode:
			w.WriteInt16(m.explodeDelay)
		case DropDestroyTypePetPickUp:
			w.WriteInt(m.characterId)
			w.WriteInt(m.petPickupExtra)
		}
		return w.Bytes()
	}
}

func (m *Destroy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.destroyType = DropDestroyType(r.ReadByte())
		m.dropId = r.ReadUint32()
		switch m.destroyType {
		case DropDestroyTypePickUp, DropDestroyTypeUnk1:
			m.characterId = r.ReadUint32()
		case DropDestroyTypeExplode:
			m.explodeDelay = r.ReadInt16()
		case DropDestroyTypePetPickUp:
			m.characterId = r.ReadUint32()
			m.petPickupExtra = r.ReadUint32()
		}
	}
}
