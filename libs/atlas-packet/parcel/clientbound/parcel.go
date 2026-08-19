package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
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

func (m *OpenQuick) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// The fifteen arms below are the bodyless PARCEL notice/error results
// (task-241 Task 8): CParcelDlg::NoticeResult @0x6F5BE2 (v83) drives the
// leading mode byte through a StringPool lookup and shows a text-only
// notice/error dialog — no additional bytes follow the mode. Each mode value
// is resolved per-tenant via the arm's body function in parcel_body.go
// against docs/packets/dispatchers/parcel.yaml (Task 6); none of these keys
// carry a jms_v185 value in that table (Ruling 5 — the jms_v185 notice-slot
// mapping is genuinely underdetermined and deliberately left unset, not
// guessed).

// SendEnableActions - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#SendEnableActions
type SendEnableActions struct {
	mode byte
}

func NewParcelSendEnableActions(mode byte) SendEnableActions {
	return SendEnableActions{mode: mode}
}

func (m SendEnableActions) Mode() byte        { return m.mode }
func (m SendEnableActions) Operation() string { return ParcelWriter }
func (m SendEnableActions) String() string    { return "parcel send enable actions" }

func (m SendEnableActions) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *SendEnableActions) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// NotEnoughMesos - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#NotEnoughMesos
type NotEnoughMesos struct {
	mode byte
}

func NewParcelNotEnoughMesos(mode byte) NotEnoughMesos {
	return NotEnoughMesos{mode: mode}
}

func (m NotEnoughMesos) Mode() byte        { return m.mode }
func (m NotEnoughMesos) Operation() string { return ParcelWriter }
func (m NotEnoughMesos) String() string    { return "parcel not enough mesos" }

func (m NotEnoughMesos) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *NotEnoughMesos) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// IncorrectRequest - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#IncorrectRequest
type IncorrectRequest struct {
	mode byte
}

func NewParcelIncorrectRequest(mode byte) IncorrectRequest {
	return IncorrectRequest{mode: mode}
}

func (m IncorrectRequest) Mode() byte        { return m.mode }
func (m IncorrectRequest) Operation() string { return ParcelWriter }
func (m IncorrectRequest) String() string    { return "parcel incorrect request" }

func (m IncorrectRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *IncorrectRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// NameDoesNotExist - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#NameDoesNotExist
type NameDoesNotExist struct {
	mode byte
}

func NewParcelNameDoesNotExist(mode byte) NameDoesNotExist {
	return NameDoesNotExist{mode: mode}
}

func (m NameDoesNotExist) Mode() byte        { return m.mode }
func (m NameDoesNotExist) Operation() string { return ParcelWriter }
func (m NameDoesNotExist) String() string    { return "parcel name does not exist" }

func (m NameDoesNotExist) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *NameDoesNotExist) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// SameAccount - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#SameAccount
type SameAccount struct {
	mode byte
}

func NewParcelSameAccount(mode byte) SameAccount {
	return SameAccount{mode: mode}
}

func (m SameAccount) Mode() byte        { return m.mode }
func (m SameAccount) Operation() string { return ParcelWriter }
func (m SameAccount) String() string    { return "parcel same account" }

func (m SameAccount) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *SameAccount) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ReceiverStorageFull - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#ReceiverStorageFull
type ReceiverStorageFull struct {
	mode byte
}

func NewParcelReceiverStorageFull(mode byte) ReceiverStorageFull {
	return ReceiverStorageFull{mode: mode}
}

func (m ReceiverStorageFull) Mode() byte        { return m.mode }
func (m ReceiverStorageFull) Operation() string { return ParcelWriter }
func (m ReceiverStorageFull) String() string    { return "parcel receiver storage full" }

func (m ReceiverStorageFull) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ReceiverStorageFull) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ReceiverUnableToReceive - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#ReceiverUnableToReceive
type ReceiverUnableToReceive struct {
	mode byte
}

func NewParcelReceiverUnableToReceive(mode byte) ReceiverUnableToReceive {
	return ReceiverUnableToReceive{mode: mode}
}

func (m ReceiverUnableToReceive) Mode() byte        { return m.mode }
func (m ReceiverUnableToReceive) Operation() string { return ParcelWriter }
func (m ReceiverUnableToReceive) String() string    { return "parcel receiver unable to receive" }

func (m ReceiverUnableToReceive) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ReceiverUnableToReceive) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// SenderUniqueConflict - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#SenderUniqueConflict
type SenderUniqueConflict struct {
	mode byte
}

func NewParcelSenderUniqueConflict(mode byte) SenderUniqueConflict {
	return SenderUniqueConflict{mode: mode}
}

func (m SenderUniqueConflict) Mode() byte        { return m.mode }
func (m SenderUniqueConflict) Operation() string { return ParcelWriter }
func (m SenderUniqueConflict) String() string    { return "parcel sender unique conflict" }

func (m SenderUniqueConflict) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *SenderUniqueConflict) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MesoLimit - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#MesoLimit
type MesoLimit struct {
	mode byte
}

func NewParcelMesoLimit(mode byte) MesoLimit {
	return MesoLimit{mode: mode}
}

func (m MesoLimit) Mode() byte        { return m.mode }
func (m MesoLimit) Operation() string { return ParcelWriter }
func (m MesoLimit) String() string    { return "parcel meso limit" }

func (m MesoLimit) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *MesoLimit) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// SuccessfullySent - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#SuccessfullySent
type SuccessfullySent struct {
	mode byte
}

func NewParcelSuccessfullySent(mode byte) SuccessfullySent {
	return SuccessfullySent{mode: mode}
}

func (m SuccessfullySent) Mode() byte        { return m.mode }
func (m SuccessfullySent) Operation() string { return ParcelWriter }
func (m SuccessfullySent) String() string    { return "parcel successfully sent" }

func (m SuccessfullySent) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *SuccessfullySent) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// UnknownError - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#UnknownError
type UnknownError struct {
	mode byte
}

func NewParcelUnknownError(mode byte) UnknownError {
	return UnknownError{mode: mode}
}

func (m UnknownError) Mode() byte        { return m.mode }
func (m UnknownError) Operation() string { return ParcelWriter }
func (m UnknownError) String() string    { return "parcel unknown error" }

func (m UnknownError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *UnknownError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// RecvEnableActions - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#RecvEnableActions
type RecvEnableActions struct {
	mode byte
}

func NewParcelRecvEnableActions(mode byte) RecvEnableActions {
	return RecvEnableActions{mode: mode}
}

func (m RecvEnableActions) Mode() byte        { return m.mode }
func (m RecvEnableActions) Operation() string { return ParcelWriter }
func (m RecvEnableActions) String() string    { return "parcel recv enable actions" }

func (m RecvEnableActions) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *RecvEnableActions) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// RecvNoFreeSlots - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#RecvNoFreeSlots
type RecvNoFreeSlots struct {
	mode byte
}

func NewParcelRecvNoFreeSlots(mode byte) RecvNoFreeSlots {
	return RecvNoFreeSlots{mode: mode}
}

func (m RecvNoFreeSlots) Mode() byte        { return m.mode }
func (m RecvNoFreeSlots) Operation() string { return ParcelWriter }
func (m RecvNoFreeSlots) String() string    { return "parcel recv no free slots" }

func (m RecvNoFreeSlots) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *RecvNoFreeSlots) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// RecvUniqueConflict - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#RecvUniqueConflict
type RecvUniqueConflict struct {
	mode byte
}

func NewParcelRecvUniqueConflict(mode byte) RecvUniqueConflict {
	return RecvUniqueConflict{mode: mode}
}

func (m RecvUniqueConflict) Mode() byte        { return m.mode }
func (m RecvUniqueConflict) Operation() string { return ParcelWriter }
func (m RecvUniqueConflict) String() string    { return "parcel recv unique conflict" }

func (m RecvUniqueConflict) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *RecvUniqueConflict) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// UnknownError2 - just mode byte.
//
// packet-audit:fname CParcelDlg::OnPacket#UnknownError2
type UnknownError2 struct {
	mode byte
}

func NewParcelUnknownError2(mode byte) UnknownError2 {
	return UnknownError2{mode: mode}
}

func (m UnknownError2) Mode() byte        { return m.mode }
func (m UnknownError2) Operation() string { return ParcelWriter }
func (m UnknownError2) String() string    { return "parcel unknown error 2" }

func (m UnknownError2) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *UnknownError2) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
