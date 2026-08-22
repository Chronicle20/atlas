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

const MapleLifeResultWriter = "MapleLifeResult"

// Reason-code keys for the tenant socket-config `operations` table of the
// MapleLifeResult writer. The VALUES live in the applicable seed templates,
// never here (DOM-25) — a key missing from a template resolves to
// ResolveCode's loud 99 sentinel, which as an int8 is POSITIVE (99) and so
// lands in the client's TAKEN arm, not UNKNOWN_ERROR. Callers must configure
// every key this writer uses.
//
// The enumeration is CUICharacterSaleDlg::OnCheckDuplicatedIDResult's own
// SIGNED three-way branch on nResult, decompiled this pass on gms_v83
// @0x7d768a, gms_v87 @0x82e12c, gms_v92 @0x756370 and gms_v95 @0x777e40 — all
// four byte-for-byte identical in structure (derivation.md §4.2/§4.4/§4.5/§4.6/§4.7),
// and structurally identical to the sibling
// CCashShop::OnCheckDuplicatedIDResult precedent this codec's shape is copied
// from (libs/atlas-packet/cash/clientbound/check_name_change.go):
//
//	nResult > 0   Notice(SP) — "this name is currently being used, please check again"
//	nResult == 0  Notice(SP) — enables the dialog's "next" button
//	nResult < 0   Notice(SP) — "unknown error (%d)", formatted with nResult
//
// gms_v83 @0x7d76c7 compiles this as `if (v4 <= 0) { if (v4) {...} else {...} } else {...}`
// — the signed comparison is unambiguous on every version checked.
const (
	// MapleLifeResultAvailable is the nResult == 0 arm — the name may be
	// used. The client enables the dialog's "next" button and transitions
	// via its internal OnButtonClicked(1000) UI code (not a wire value).
	MapleLifeResultAvailable = "AVAILABLE"
	// MapleLifeResultTaken is the nResult > 0 arm — "this name is currently
	// being used, please check again." ANY positive byte value satisfies the
	// client's signed `> 0` test; the templates configure 1 as the canonical
	// representative.
	MapleLifeResultTaken = "TAKEN"
	// MapleLifeResultUnknownError is the nResult < 0 arm — "unknown error
	// (%d)", formatted with nResult. ANY negative byte value (when the wire
	// byte is reinterpreted as signed) satisfies the client's `< 0` test; the
	// templates configure 0xFF (-1 as int8) as the canonical representative.
	MapleLifeResultUnknownError = "UNKNOWN_ERROR"
)

// MapleLifeResult is the clientbound MAPLELIFE_RESULT op —
// CUICharacterSaleDlg::OnCheckDuplicatedIDResult, the server's answer to the
// client's duplicate-name-check request issued while composing a Maple Life
// (Cash/0543) character-slot purchase.
//
// Body, IDENTICAL on every in-scope version — gms_v83, gms_v87, gms_v92,
// gms_v95 (derivation.md §4.7; gms_v84 is VERSION-ABSENT — no
// CUICharacterSaleDlg code path exists on that binary, §4.3):
//
//	DecodeStr  sName    // echoed back to the client, formatting/UI only
//	Decode1    nResult  // SIGNED: >0 taken, ==0 available, <0 unknown error
//
// There is deliberately NO version gate in Encode/Decode: no field, width or
// order differs across the four in-scope versions. The only per-version
// value is the opcode, which lives in the tenant template.
//
// Receiver addresses (derivation.md §4.2/§4.4/§4.5/§4.6, decompiled this
// pass): gms_v83 0x7d768a, gms_v87 0x82e12c, gms_v92 0x756370,
// gms_v95 0x777e40. gms_v84: VERSION-ABSENT (§4.3).
//
// Derivation: docs/tasks/task-246-maple-life-character-creation/derivation.md §4.
// packet-audit:fname CUICharacterSaleDlg::OnCheckDuplicatedIDResult
type MapleLifeResult struct {
	name   string
	result int8
}

func NewMapleLifeResult(name string, result int8) MapleLifeResult {
	return MapleLifeResult{name: name, result: result}
}

// Name is the character name the client is checking. Echoed back; decoded
// but only used for formatting/UI on the client, not a routing key.
func (m MapleLifeResult) Name() string { return m.name }

// Result is the resolved wire byte, NOT a semantic code, reinterpreted as
// signed to match the client's comparison. It comes from the tenant
// template's operations table via one of the body builders below.
func (m MapleLifeResult) Result() int8 { return m.result }

func (m MapleLifeResult) Operation() string {
	return MapleLifeResultWriter
}

func (m MapleLifeResult) String() string {
	return fmt.Sprintf("name [%s], result [%d]", m.name, m.result)
}

func (m MapleLifeResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		w.WriteInt8(m.result)
		return w.Bytes()
	}
}

func (m *MapleLifeResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		m.result = r.ReadInt8()
	}
}

// MapleLifeResultBody emits an arbitrary arm by its operations-table key —
// key must be one of the MapleLifeResult* constants; anything else resolves
// to ResolveCode's 99 sentinel, which as an int8 (99) is positive and lands
// in the client's TAKEN arm — not UNKNOWN_ERROR — so callers should route
// through MapleLifeResultRejectedBody rather than an unrecognised key here.
func MapleLifeResultBody(name string, key string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", key, func(code byte) packet.Encoder {
		return NewMapleLifeResult(name, int8(code))
	})
}

// MapleLifeResultRejectedBody emits a refusal arm for a server-side
// name-validity reason (character.NameReason* on the atlas-channel side —
// libs/atlas-packet must not import atlas-channel, so the mapping below is
// keyed on the same literal strings: "length", "regex", "duplicate",
// "reserved"). Only "duplicate" lands on a semantically exact arm
// (nResult > 0 — the client's own "this name is currently being used"
// string); the other three collapse onto UNKNOWN_ERROR because no arm of
// OnCheckDuplicatedIDResult renders a length/charset/reserved-specific
// string on any version examined (derivation.md §4.2/§4.4/§4.5/§4.6).
func MapleLifeResultRejectedBody(name string, reason string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MapleLifeResultBody(name, mapleLifeResultReasonKey(reason))
}

// mapleLifeResultReasonKey maps a server-side name-validity reason onto the
// operations-table key whose configured byte this client version renders.
func mapleLifeResultReasonKey(reason string) string {
	if key, ok := mapleLifeResultReasonArms[reason]; ok {
		return key
	}
	return MapleLifeResultUnknownError
}

// mapleLifeResultReasonArms is the reason -> arm table as DATA, so a test can
// assert it covers the four reasons character.NameReason* enumerates
// (services/atlas-channel/atlas.com/channel/character/name_validity_requests.go)
// without libs/atlas-packet importing an atlas-channel package.
var mapleLifeResultReasonArms = map[string]string{
	"length":    MapleLifeResultUnknownError,
	"regex":     MapleLifeResultUnknownError,
	"duplicate": MapleLifeResultTaken,
	"reserved":  MapleLifeResultUnknownError,
}
