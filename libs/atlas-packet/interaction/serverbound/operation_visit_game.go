package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// packet-audit:fname CUserLocal::HandleLButtonDblClk
//
// ida-notes.md §G4 (v83 @ 0x94fbbf): double-click on a miniroom balloon sends
// mode 4 (ENTER/VISIT) as int32 serialNumber, byte hasPassword, an optional
// password string when hasPassword != 0, and a constant trailing byte 0. This
// is the shape that operation_visit.go does NOT decode (that file is the
// trade-shaped clientbound EnterResult decoder, mis-homed under serverbound —
// see Task 6 note in §G4); this type is the game-room join send.
type OperationVisitGame struct {
	serialNumber uint32
	hasPassword  bool
	password     string
}

func (m OperationVisitGame) SerialNumber() uint32 { return m.serialNumber }
func (m OperationVisitGame) HasPassword() bool    { return m.hasPassword }
func (m OperationVisitGame) Password() string     { return m.password }

func (m OperationVisitGame) Operation() string { return "OperationVisitGame" }

func (m OperationVisitGame) String() string {
	return fmt.Sprintf("serialNumber [%d] hasPassword [%t] password [%s]", m.serialNumber, m.hasPassword, m.password)
}

func (m OperationVisitGame) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.serialNumber)
		w.WriteBool(m.hasPassword)
		if m.hasPassword {
			w.WriteAsciiString(m.password)
		}
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *OperationVisitGame) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumber = r.ReadUint32()
		m.hasPassword = r.ReadBool()
		if m.hasPassword {
			m.password = r.ReadAsciiString()
		}
		r.ReadByte()
	}
}
