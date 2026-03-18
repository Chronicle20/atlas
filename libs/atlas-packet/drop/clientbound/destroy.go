package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
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

type Destroy struct {
	dropId      uint32
	destroyType DropDestroyType
	characterId uint32
	petSlot     int8
}

func NewDropDestroy(dropId uint32, destroyType DropDestroyType, characterId uint32, petSlot int8) Destroy {
	return Destroy{dropId: dropId, destroyType: destroyType, characterId: characterId, petSlot: petSlot}
}

func (m Destroy) DropId() uint32             { return m.dropId }
func (m Destroy) DestroyType() DropDestroyType { return m.destroyType }
func (m Destroy) CharacterId() uint32        { return m.characterId }
func (m Destroy) PetSlot() int8              { return m.petSlot }
func (m Destroy) Operation() string          { return DropDestroyWriter }
func (m Destroy) String() string {
	return fmt.Sprintf("dropId [%d], destroyType [%d]", m.dropId, m.destroyType)
}

func (m Destroy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.destroyType))
		w.WriteInt(m.dropId)
		if m.destroyType >= 2 {
			w.WriteInt(m.characterId)
			if m.petSlot >= 0 {
				w.WriteByte(byte(m.petSlot))
			}
		}
		return w.Bytes()
	}
}

func (m *Destroy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.destroyType = DropDestroyType(r.ReadByte())
		m.dropId = r.ReadUint32()
		if m.destroyType >= 2 {
			m.characterId = r.ReadUint32()
			if r.Available() > 0 {
				m.petSlot = r.ReadInt8()
			} else {
				m.petSlot = -1
			}
		} else {
			m.petSlot = -1
		}
	}
}
