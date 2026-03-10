package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetActivatedWriter = "PetActivated"

type Activated struct {
	ownerId    uint32
	slot       int8
	active     bool
	templateId uint32
	name       string
	petId      uint64
	x          int16
	y          int16
	stance     byte
	foothold   uint16
	nameTag    byte
	chatBalloon byte
	despawnMode byte
}

func NewPetSpawnActivated(ownerId uint32, slot int8, templateId uint32, name string, petId uint64, x int16, y int16, stance byte, foothold uint16) Activated {
	return Activated{
		ownerId: ownerId, slot: slot, active: true,
		templateId: templateId, name: name, petId: petId,
		x: x, y: y, stance: stance, foothold: foothold,
	}
}

func NewPetDespawnActivated(ownerId uint32, slot int8, despawnMode byte) Activated {
	return Activated{ownerId: ownerId, slot: slot, active: false, despawnMode: despawnMode}
}

func (m Activated) Operation() string { return PetActivatedWriter }
func (m Activated) String() string {
	return fmt.Sprintf("ownerId [%d], slot [%d], active [%t]", m.ownerId, m.slot, m.active)
}

func (m Activated) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt8(m.slot)
		w.WriteBool(m.active)
		if m.active {
			w.WriteBool(true) // show
			w.WriteInt(m.templateId)
			w.WriteAsciiString(m.name)
			w.WriteLong(m.petId)
			w.WriteInt16(m.x)
			w.WriteInt16(m.y)
			w.WriteByte(m.stance)
			w.WriteShort(m.foothold)
			w.WriteByte(m.nameTag)
			w.WriteByte(m.chatBalloon)
		} else {
			w.WriteByte(m.despawnMode)
		}
		return w.Bytes()
	}
}

func (m *Activated) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.slot = r.ReadInt8()
		m.active = r.ReadBool()
		if m.active {
			_ = r.ReadBool() // show
			m.templateId = r.ReadUint32()
			m.name = r.ReadAsciiString()
			m.petId = r.ReadUint64()
			m.x = r.ReadInt16()
			m.y = r.ReadInt16()
			m.stance = r.ReadByte()
			m.foothold = r.ReadUint16()
			m.nameTag = r.ReadByte()
			m.chatBalloon = r.ReadByte()
		} else {
			m.despawnMode = r.ReadByte()
		}
	}
}
