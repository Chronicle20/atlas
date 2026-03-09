package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const PetDropPickUpHandle = "PetDropPickUpHandle"

type DropPickUp struct {
	petId          uint64
	fieldKey       byte
	updateTime     uint32
	x              int16
	y              int16
	dropId         uint32
	crc            uint32
	bPickupOthers  bool
	bSweepForDrop  bool
	bLongRange     bool
	ownerX         int16
	ownerY         int16
	posCrc         uint32
	rectCrc        uint32
}

func (m DropPickUp) PetId() uint64        { return m.petId }
func (m DropPickUp) FieldKey() byte        { return m.fieldKey }
func (m DropPickUp) UpdateTime() uint32    { return m.updateTime }
func (m DropPickUp) X() int16              { return m.x }
func (m DropPickUp) Y() int16              { return m.y }
func (m DropPickUp) DropId() uint32        { return m.dropId }
func (m DropPickUp) Crc() uint32           { return m.crc }
func (m DropPickUp) BPickupOthers() bool   { return m.bPickupOthers }
func (m DropPickUp) BSweepForDrop() bool   { return m.bSweepForDrop }
func (m DropPickUp) BLongRange() bool      { return m.bLongRange }
func (m DropPickUp) OwnerX() int16         { return m.ownerX }
func (m DropPickUp) OwnerY() int16         { return m.ownerY }
func (m DropPickUp) PosCrc() uint32        { return m.posCrc }
func (m DropPickUp) RectCrc() uint32       { return m.rectCrc }

func (m DropPickUp) Operation() string {
	return PetDropPickUpHandle
}

func (m DropPickUp) String() string {
	return fmt.Sprintf("petId [%d] fieldKey [%d] updateTime [%d] x [%d] y [%d] dropId [%d] crc [%d] bPickupOthers [%t] bSweepForDrop [%t] bLongRange [%t]", m.petId, m.fieldKey, m.updateTime, m.x, m.y, m.dropId, m.crc, m.bPickupOthers, m.bSweepForDrop, m.bLongRange)
}

func (m DropPickUp) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.petId)
		w.WriteByte(m.fieldKey)
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteInt(m.dropId)
		w.WriteInt(m.crc)
		w.WriteBool(m.bPickupOthers)
		w.WriteBool(m.bSweepForDrop)
		w.WriteBool(m.bLongRange)
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			if m.dropId%13 != 0 {
				w.WriteInt16(m.ownerX)
				w.WriteInt16(m.ownerY)
				w.WriteInt(m.posCrc)
				w.WriteInt(m.rectCrc)
			}
		}
		return w.Bytes()
	}
}

func (m *DropPickUp) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.petId = r.ReadUint64()
		m.fieldKey = r.ReadByte()
		m.updateTime = r.ReadUint32()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.dropId = r.ReadUint32()
		m.crc = r.ReadUint32()
		m.bPickupOthers = r.ReadBool()
		m.bSweepForDrop = r.ReadBool()
		m.bLongRange = r.ReadBool()
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			if m.dropId%13 != 0 {
				m.ownerX = r.ReadInt16()
				m.ownerY = r.ReadInt16()
				m.posCrc = r.ReadUint32()
				m.rectCrc = r.ReadUint32()
			}
		}
	}
}
