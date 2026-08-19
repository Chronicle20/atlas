package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ActionSend is the DUEY_ACTION SEND arm (mode 2, jms_v185 mode 3). The body
// is asymmetric across the two client call sites (design.md §5.4, task-241
// design §0 finding C): the NPC path stops after the quick flag — no message,
// no ticket reference (v83 CTabSend::SendParcel @0x6F36A8, quick==false).
// Only the quick-send path (v83 @0x6F1DF5, quick==true) appends the trailing
// string message and uint32 ticketRef. Decode must gate those two trailing
// fields on the quick flag it just read — reading them unconditionally
// desynchronises the reader on every NPC send.
//
// packet-audit:fname CTabSend::SendParcel
type ActionSend struct {
	inventoryType byte
	slot          uint16
	quantity      uint16
	mesos         uint32
	recipientName string
	quick         bool
	message       string
	ticketRef     uint32
}

func (m ActionSend) InventoryType() byte   { return m.inventoryType }
func (m ActionSend) Slot() uint16          { return m.slot }
func (m ActionSend) Quantity() uint16      { return m.quantity }
func (m ActionSend) Mesos() uint32         { return m.mesos }
func (m ActionSend) RecipientName() string { return m.recipientName }
func (m ActionSend) Quick() bool           { return m.quick }
func (m ActionSend) Message() string       { return m.message }
func (m ActionSend) TicketRef() uint32     { return m.ticketRef }

func (m ActionSend) Operation() string {
	return "ActionSend"
}

func (m ActionSend) String() string {
	return fmt.Sprintf("inventoryType [%d] slot [%d] quantity [%d] mesos [%d] recipientName [%s] quick [%t] message [%s] ticketRef [%d]",
		m.inventoryType, m.slot, m.quantity, m.mesos, m.recipientName, m.quick, m.message, m.ticketRef)
}

func (m ActionSend) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.inventoryType)
		w.WriteShort(m.slot)
		w.WriteShort(m.quantity)
		w.WriteInt(m.mesos)
		w.WriteAsciiString(m.recipientName)
		w.WriteBool(m.quick)
		if m.quick {
			w.WriteAsciiString(m.message)
			w.WriteInt(m.ticketRef)
		}
		return w.Bytes()
	}
}

func (m *ActionSend) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.inventoryType = r.ReadByte()
		m.slot = r.ReadUint16()
		m.quantity = r.ReadUint16()
		m.mesos = r.ReadUint32()
		m.recipientName = r.ReadAsciiString()
		m.quick = r.ReadBool()
		if m.quick {
			m.message = r.ReadAsciiString()
			m.ticketRef = r.ReadUint32()
		}
	}
}
