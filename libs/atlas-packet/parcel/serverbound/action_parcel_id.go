package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ActionReceive is the DUEY_ACTION RECEIVE arm (mode 4, jms_v185 mode 5):
// uint32 parcelId only (v83 CTabReceive::ReceiveParcel @0x6F0CA3).
//
// packet-audit:fname CTabReceive::ReceiveParcel
type ActionReceive struct {
	parcelId uint32
}

func (m ActionReceive) ParcelId() uint32 { return m.parcelId }

func (m ActionReceive) Operation() string {
	return "ActionReceive"
}

func (m ActionReceive) String() string {
	return fmt.Sprintf("parcelId [%d]", m.parcelId)
}

func (m ActionReceive) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.parcelId)
		return w.Bytes()
	}
}

func (m *ActionReceive) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.parcelId = r.ReadUint32()
	}
}

// ActionDiscard is the DUEY_ACTION DISCARD arm (mode 5, jms_v185 mode 6):
// uint32 parcelId only (v83 CTabReceive::DiscardParcel @0x6F0DC3). Wire-shape
// identical to ActionReceive but kept as a separate struct — the two arms
// are independent client call sites with independent modes.
//
// packet-audit:fname CTabReceive::DiscardParcel
type ActionDiscard struct {
	parcelId uint32
}

func (m ActionDiscard) ParcelId() uint32 { return m.parcelId }

func (m ActionDiscard) Operation() string {
	return "ActionDiscard"
}

func (m ActionDiscard) String() string {
	return fmt.Sprintf("parcelId [%d]", m.parcelId)
}

func (m ActionDiscard) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.parcelId)
		return w.Bytes()
	}
}

func (m *ActionDiscard) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.parcelId = r.ReadUint32()
	}
}

// ActionClose is the DUEY_ACTION CLOSE arm (mode 7, jms_v185 mode 8): no
// body (v83 CParcelDlg::CloseParcelDlg @0x6F5691).
//
// packet-audit:fname CParcelDlg::CloseParcelDlg
type ActionClose struct{}

func (m ActionClose) Operation() string {
	return "ActionClose"
}

func (m ActionClose) String() string {
	return "ActionClose"
}

func (m ActionClose) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *ActionClose) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {}
}
