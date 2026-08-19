package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ParcelWriter is the Parcel dispatcher's writer/operation name
// (docs/packets/dispatchers/parcel.yaml, fname CParcelDlg::OnPacket).
const ParcelWriter = "Parcel"

// Open - mode, quickEnabled, mailbox parcels, newly-arrived parcels.
//
// CTabReceive::SetParcel @0x6EF69C: bool quickEnabled (Decode1), byte
// count + count*PARCEL::Decode (the mailbox), byte newCount +
// newCount*PARCEL::Decode (parcels arrived since the last open — each one
// separately raised as a CUtilDlg::Notice, design.md §5.3).
//
// packet-audit:fname CParcelDlg::OnPacket#Open
type Open struct {
	mode         byte
	quickEnabled bool
	mailbox      []parcel.Parcel
	arrived      []parcel.Parcel
}

func NewParcelOpen(mode byte, quickEnabled bool, mailbox []parcel.Parcel, arrived []parcel.Parcel) Open {
	return Open{mode: mode, quickEnabled: quickEnabled, mailbox: mailbox, arrived: arrived}
}

func (m Open) Mode() byte               { return m.mode }
func (m Open) QuickEnabled() bool       { return m.quickEnabled }
func (m Open) Mailbox() []parcel.Parcel { return m.mailbox }
func (m Open) Arrived() []parcel.Parcel { return m.arrived }
func (m Open) Operation() string        { return ParcelWriter }

func (m Open) String() string {
	return fmt.Sprintf("parcel open mailbox [%d] arrived [%d]", len(m.mailbox), len(m.arrived))
}

func (m Open) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.quickEnabled)
		w.WriteByte(byte(len(m.mailbox)))
		for _, p := range m.mailbox {
			w.WriteByteArray(p.Encode(l, ctx)(options))
		}
		w.WriteByte(byte(len(m.arrived)))
		for _, p := range m.arrived {
			w.WriteByteArray(p.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

// OpenQuick - just mode byte.
//
// CParcelDlg::OnPacket mode 0x1A (design.md §5.2 OPEN_QUICK): no additional
// bytes are read; it re-enables the quick-send controls client-side.
//
// packet-audit:fname CParcelDlg::OnPacket#OpenQuick
type OpenQuick struct {
	mode byte
}

func NewParcelOpenQuick(mode byte) OpenQuick {
	return OpenQuick{mode: mode}
}

func (m OpenQuick) Mode() byte        { return m.mode }
func (m OpenQuick) Operation() string { return ParcelWriter }
func (m OpenQuick) String() string    { return "parcel open quick" }

func (m OpenQuick) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}
