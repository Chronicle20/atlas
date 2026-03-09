package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterInventoryMoveHandle = "CharacterInventoryMoveHandle"

// Move - CWvsContext::SendGatherItemRequest / CWvsContext::SendSortItemRequest
type Move struct {
	updateTime    uint32
	inventoryType byte
	source        int16
	destination   int16
	count         int16
}

func (m Move) UpdateTime() uint32    { return m.updateTime }
func (m Move) InventoryType() byte   { return m.inventoryType }
func (m Move) Source() int16         { return m.source }
func (m Move) Destination() int16    { return m.destination }
func (m Move) Count() int16          { return m.count }

func (m Move) Operation() string {
	return CharacterInventoryMoveHandle
}

func (m Move) String() string {
	return fmt.Sprintf("updateTime [%d], inventoryType [%d], source [%d], destination [%d], count [%d]", m.updateTime, m.inventoryType, m.source, m.destination, m.count)
}

func (m Move) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.source)
		w.WriteInt16(m.destination)
		w.WriteInt16(m.count)
		return w.Bytes()
	}
}

func (m *Move) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.inventoryType = r.ReadByte()
		m.source = r.ReadInt16()
		m.destination = r.ReadInt16()
		m.count = r.ReadInt16()
	}
}
