package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationCreate struct {
	roomType  byte
	title     string
	private   bool
	password  string
	nGameSpec byte
	slot      int16
	itemId    uint32
}

func (m OperationCreate) RoomType() byte   { return m.roomType }
func (m OperationCreate) Title() string    { return m.title }
func (m OperationCreate) Private() bool    { return m.private }
func (m OperationCreate) Password() string { return m.password }
func (m OperationCreate) NGameSpec() byte  { return m.nGameSpec }
func (m OperationCreate) Slot() int16      { return m.slot }
func (m OperationCreate) ItemId() uint32   { return m.itemId }

func (m OperationCreate) Operation() string { return "OperationCreate" }

func (m OperationCreate) String() string {
	return fmt.Sprintf("roomType [%d] title [%s] private [%t] password [%s] nGameSpec [%d] slot [%d] itemId [%d]", m.roomType, m.title, m.private, m.password, m.nGameSpec, m.slot, m.itemId)
}

func (m OperationCreate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.roomType)
		if m.roomType == 1 || m.roomType == 2 {
			w.WriteAsciiString(m.title)
			w.WriteBool(m.private)
			if m.private {
				w.WriteAsciiString(m.password)
			}
			w.WriteByte(m.nGameSpec)
		} else if m.roomType == 3 {
			w.WriteBool(m.private)
		} else if m.roomType == 4 || m.roomType == 5 {
			w.WriteAsciiString(m.title)
			w.WriteBool(m.private)
			w.WriteInt16(m.slot)
			w.WriteInt(m.itemId)
		} else if m.roomType == 6 {
			w.WriteBool(m.private)
		}
		return w.Bytes()
	}
}

func (m *OperationCreate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.roomType = r.ReadByte()
		if m.roomType == 1 || m.roomType == 2 {
			m.title = r.ReadAsciiString()
			m.private = r.ReadBool()
			if m.private {
				m.password = r.ReadAsciiString()
			}
			m.nGameSpec = r.ReadByte()
		} else if m.roomType == 3 {
			m.private = r.ReadBool()
		} else if m.roomType == 4 || m.roomType == 5 {
			m.title = r.ReadAsciiString()
			m.private = r.ReadBool()
			m.slot = r.ReadInt16()
			m.itemId = r.ReadUint32()
		} else if m.roomType == 6 {
			m.private = r.ReadBool()
		}
	}
}
