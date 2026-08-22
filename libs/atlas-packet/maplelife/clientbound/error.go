package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MapleLifeErrorWriter = "MapleLifeError"

// Arm-key constants for the tenant socket-config `operations` table of the
// MapleLifeErrorWriter. The VALUES (the raw `nType` literal each version's
// client compares against) live in the applicable seed templates, never here
// (DOM-25) — a key missing from a template resolves to ResolveCode's loud 99
// sentinel. Callers must configure every key this writer uses, on every
// in-scope version.
//
// The enumeration is CUICharacterSaleDlg::OnCreateNewCharacterResult's own
// exact-equality switch on `nType` (NOT a range/signed test), decompiled this
// pass on gms_v83 @0x7d77b0, gms_v87 @0x82e252, gms_v92 @0x7564f0, and
// gms_v95 @0x777fc0 — identical decode order and branch SHAPE on all four;
// only the raw `nType` literal per arm shifts per version
// (derivation.md §5.1-§5.5):
//
//	nType == SUCCESS literal && nParam == 0   Notice(SP) — creation succeeded
//	nType == SUCCESS literal && nParam != 0   Notice(SP) — "unknown error (%d)", formatted with nParam
//	nType == NAME_TAKEN_AT_SUBMIT literal      Notice(SP) — "you can not use this name, please check again"
//	any other nType                            Notice(SP) — "unknown error (%d)", formatted with nParam
//
// This is a closed THREE-semantic-arm enumeration — SUCCESS,
// NAME_TAKEN_AT_SUBMIT, UNKNOWN_ERROR(param) — even though the client's raw
// switch has four textual branches: the SUCCESS literal's nParam!=0 sub-case
// and the switch's default case both render the identical
// "unknown error (%d)" string and both carry nParam as the formatted
// diagnostic value, so both collapse onto the UNKNOWN_ERROR arm here. No
// design-anticipated "invalid look" arm was found on any version — the two
// rejectable-at-server outcomes the client renders on the create-character
// path are duplicate-name-at-submit and the generic unknown-error fallback
// (derivation.md §5.5). The SUCCESS arm ships through THIS op, not through
// MAPLELIFE_RESULT — design §5.4 — so a reader does not mistake this writer
// name for a failure-only channel.
const (
	// MapleLifeErrorSuccess is the nType==SUCCESS-literal && nParam==0 arm —
	// character creation completed successfully.
	MapleLifeErrorSuccess = "SUCCESS"
	// MapleLifeErrorNameTakenAtSubmit is the nType==NAME_TAKEN_AT_SUBMIT-literal
	// arm — the name was rejected as a duplicate at submit time (distinct
	// from the pre-submit duplicate-name-check answer, MAPLELIFE_RESULT).
	MapleLifeErrorNameTakenAtSubmit = "NAME_TAKEN_AT_SUBMIT"
	// MapleLifeErrorUnknownError is the generic fallback arm — any nType not
	// recognised as SUCCESS or NAME_TAKEN_AT_SUBMIT, and the SUCCESS
	// literal's nParam!=0 sub-case. nParam is carried through as a formatted
	// diagnostic value, not a fixed enum member.
	MapleLifeErrorUnknownError = "UNKNOWN_ERROR"
)

// MapleLifeError is the clientbound MAPLELIFE_ERROR op —
// CUICharacterSaleDlg::OnCreateNewCharacterResult, the server's answer to the
// client's SendCreateNewCharacter submit for a Maple Life (Cash/0543)
// character-slot purchase.
//
// Body, IDENTICAL SHAPE on every in-scope version — gms_v83, gms_v87, gms_v92,
// gms_v95 (derivation.md §5.5; gms_v84 is VERSION-ABSENT — no
// CUICharacterSaleDlg code path exists on that binary):
//
//	Decode1  nType   // arm selector; per-version literal, tenant-template config (DOM-25)
//	Decode4  nParam  // diagnostic value, formatted into the UNKNOWN_ERROR string only
//
// There is deliberately NO version gate in Encode/Decode: no field, width or
// order differs across the four in-scope versions. Only the opcode and the
// per-version `nType` literal per arm are tenant-template config, never a Go
// literal or a code branch.
//
// Receiver addresses (derivation.md §5.1-§5.4, decompiled this pass):
// gms_v83 0x7d77b0, gms_v87 0x82e252, gms_v92 0x7564f0, gms_v95 0x777fc0.
// gms_v84: VERSION-ABSENT.
//
// Derivation: docs/tasks/task-246-maple-life-character-creation/derivation.md §5.
// packet-audit:fname CUICharacterSaleDlg::OnCreateNewCharacterResult
type MapleLifeError struct {
	nType  byte
	nParam uint32
}

func NewMapleLifeError(nType byte, nParam uint32) MapleLifeError {
	return MapleLifeError{nType: nType, nParam: nParam}
}

// NType is the resolved wire byte selecting which arm the client renders. It
// comes from the tenant template's operations table via one of the body
// builders below, not a fixed Go value.
func (m MapleLifeError) NType() byte { return m.nType }

// NParam is the diagnostic value formatted into the client's
// "unknown error (%d)" string on the UNKNOWN_ERROR arm; ignored (but still
// present on the wire) on the other two arms.
func (m MapleLifeError) NParam() uint32 { return m.nParam }

func (m MapleLifeError) Operation() string {
	return MapleLifeErrorWriter
}

func (m MapleLifeError) String() string {
	return fmt.Sprintf("nType [%d], nParam [%d]", m.nType, m.nParam)
}

func (m MapleLifeError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.nType)
		w.WriteInt(m.nParam)
		return w.Bytes()
	}
}

func (m *MapleLifeError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.nType = r.ReadByte()
		m.nParam = r.ReadUint32()
	}
}

// MapleLifeErrorBody emits an arbitrary arm by its operations-table key — key
// must be one of the MapleLifeError* constants; anything else resolves to
// ResolveCode's 99 sentinel, which the client has no dedicated arm for.
// nParam is always sent as 0 — the canonical representative for every arm;
// SUCCESS and NAME_TAKEN_AT_SUBMIT never inspect nParam, and UNKNOWN_ERROR's
// diagnostic value is not yet threaded through this helper.
func MapleLifeErrorBody(key string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", key, func(code byte) packet.Encoder {
		return NewMapleLifeError(code, 0)
	})
}
