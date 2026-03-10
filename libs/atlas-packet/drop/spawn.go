package drop

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const DropSpawnWriter = "DropSpawn"

type DropEnterType byte

const (
	DropEnterTypeFresh     DropEnterType = 1
	DropEnterTypeExisting  DropEnterType = 2
	DropEnterTypeDisappear DropEnterType = 3
)

type Spawn struct {
	enterType     DropEnterType
	dropId        uint32
	meso          uint32
	itemId        uint32
	owner         uint32
	dropType      byte
	x             int16
	y             int16
	dropperId     uint32
	dropperX      int16
	dropperY      int16
	delay         int16
	characterDrop bool
}

func NewDropSpawn(enterType DropEnterType, dropId uint32, meso uint32, itemId uint32, owner uint32, dropType byte, x int16, y int16, dropperId uint32, dropperX int16, dropperY int16, delay int16, characterDrop bool) Spawn {
	return Spawn{
		enterType: enterType, dropId: dropId, meso: meso, itemId: itemId,
		owner: owner, dropType: dropType, x: x, y: y, dropperId: dropperId,
		dropperX: dropperX, dropperY: dropperY, delay: delay, characterDrop: characterDrop,
	}
}

func (m Spawn) Operation() string { return DropSpawnWriter }
func (m Spawn) String() string {
	return fmt.Sprintf("dropId [%d], enterType [%d], meso [%d], itemId [%d]", m.dropId, m.enterType, m.meso, m.itemId)
}

func (m Spawn) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.enterType))
		w.WriteInt(m.dropId)
		if m.meso > 0 {
			w.WriteBool(true)
			w.WriteInt(m.meso)
		} else {
			w.WriteBool(false)
			w.WriteInt(m.itemId)
		}
		w.WriteInt(m.owner)
		w.WriteByte(m.dropType)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteInt(m.dropperId)
		if m.enterType != 2 {
			w.WriteInt16(m.dropperX)
			w.WriteInt16(m.dropperY)
			w.WriteInt16(m.delay)
		}
		if m.meso == 0 {
			w.WriteInt64(-1)
		}
		w.WriteBool(!m.characterDrop)
		return w.Bytes()
	}
}

func (m *Spawn) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.enterType = DropEnterType(r.ReadByte())
		m.dropId = r.ReadUint32()
		isMeso := r.ReadBool()
		if isMeso {
			m.meso = r.ReadUint32()
		} else {
			m.itemId = r.ReadUint32()
		}
		m.owner = r.ReadUint32()
		m.dropType = r.ReadByte()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.dropperId = r.ReadUint32()
		if m.enterType != 2 {
			m.dropperX = r.ReadInt16()
			m.dropperY = r.ReadInt16()
			m.delay = r.ReadInt16()
		}
		if !isMeso {
			_ = r.ReadInt64() // expiration
		}
		m.characterDrop = !r.ReadBool()
	}
}
