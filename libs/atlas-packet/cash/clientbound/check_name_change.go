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

const CashShopCheckNameChangeWriter = "CashShopCheckNameChange"

// Reason-code keys for the tenant socket-config `operations` table of the
// CashShopCheckNameChange writer. The VALUES live in the nine applicable seed
// templates, never here (DOM-25) — a key that is missing from a template
// resolves to ResolveCode's loud 99 sentinel, which as an int8 is POSITIVE
// (99) and so lands in the client's TAKEN arm, not UNKNOWN_ERROR. Callers
// must configure every key this writer uses; do not rely on the sentinel to
// fail safely here the way it does on ops whose default arm is negative.
//
// The enumeration is CCashShop::OnCheckDuplicatedIDResult's own SIGNED
// three-way branch on nResult, independently re-decompiled this pass on
// gms_v48 @0x455a7f, gms_v61 @0x463900, gms_v72 @0x473519, gms_v83 @0x47baea
// and gms_v95 @0x497fb0 — all five byte-for-byte identical in structure
// (derivation.md §2.3/§4.4):
//
//	nResult > 0   Notice(SP) — "this name is currently in use" (name taken)
//	nResult == 0  Notice(SP) — "this name can be used" + CUIChangingCharacterName::SetNameValues(sName)
//	nResult < 0   Notice(SP) — "an unknown error has occurred"
//
// gms_v95 @0x498019 compiles this as `if (v3 <= 0) { if (!v3) {...} } else {...}`
// — the signed comparison is unambiguous in every version checked.
const (
	// CheckNameChangeAvailable is the nResult == 0 arm — the name may be used.
	// The client stashes sName into the rename dialog via
	// CUIChangingCharacterName::SetNameValues (gms_v95 @0x49805e).
	CheckNameChangeAvailable = "AVAILABLE"
	// CheckNameChangeTaken is the nResult > 0 arm — "this name is currently
	// being used, please use another name." ANY positive byte value satisfies
	// the client's signed `> 0` test; the templates configure 1 as the
	// canonical representative.
	CheckNameChangeTaken = "TAKEN"
	// CheckNameChangeUnknownError is the nResult < 0 arm — "an unknown error
	// has occurred." ANY negative byte value (when the wire byte is
	// reinterpreted as signed) satisfies the client's `< 0` test; the
	// templates configure 0xFF (-1 as int8) as the canonical representative.
	CheckNameChangeUnknownError = "UNKNOWN_ERROR"
)

// CheckNameChange is the clientbound CASHSHOP_CHECK_NAME_CHANGE op —
// CCashShop::OnCheckDuplicatedIDResult, the server's answer to the client's
// duplicate-name-check request issued while composing a NAME_TRANSFER
// purchase in the cash shop rename dialog.
//
// Body, IDENTICAL on every applicable version — v48, v61, v72, v79, v83, v84,
// v87, v92, v95 (derivation.md §2.3, independently re-decompiled this pass on
// v48/v61/v72/v83/v95; v79/v84/v87/v92 confirmed structurally identical by
// derivation.md's prior pass and left unquestioned here since none of the
// five directly re-checked versions shows any divergence a middle version
// could plausibly introduce):
//
//	DecodeStr  sName
//	Decode1    nResult   // SIGNED: >0 taken, ==0 available, <0 unknown error
//
// There is deliberately NO version gate in Encode/Decode: no field, width or
// order differs across the nine versions. The only per-version value is the
// opcode, which lives in the tenant template.
//
// Receiver addresses (derivation.md §2.3, re-confirmed this pass for the
// first five): v48 0x455a7f, v61 0x463900, v72 0x473519, v79 0x4749e5,
// v83 0x47baea, v84 0x47ec88, v87 0x4872cb, v92 0x493f40, v95 0x497fb0.
// jms_v185: op absent (no name-change feature at all — derivation.md §1.5).
//
// # This row owns the name-validity reason taxonomy — and cannot fully carry it
//
// design.md §6 defines a closed four-reason taxonomy for a rejected rename:
// name_taken, name_reserved, name_invalid_length, name_invalid_charset.
// CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT (cash/clientbound.CheckNameChangePossibleResult)
// cannot express any of the four — its four arms are ALLOWED /
// ALREADY_SUBMITTED / two REQUEST_LIMIT variants, none of which is a
// name-validity message (see that type's doc comment). This op is the
// candidate handoff: OnCheckDuplicatedIDResult IS the client's per-name
// answer.
//
// It only PARTIALLY succeeds. The client's switch here is a THREE-way branch,
// not four, and none of the three arms is reason-specific beyond "taken":
//
//	name_taken            -> TAKEN           (nResult > 0 — exact semantic match:
//	                                           "this name is currently in use")
//	name_reserved         -> UNKNOWN_ERROR    (no arm renders "this name is
//	                                           reserved"; the only two non-error
//	                                           arms are "available" and "taken")
//	name_invalid_length   -> UNKNOWN_ERROR    (same — no length-specific string
//	                                           exists in this handler)
//	name_invalid_charset  -> UNKNOWN_ERROR    (same — no charset-specific string
//	                                           exists in this handler)
//
// This is the finding this row was dispatched to surface: the four-reason
// taxonomy does NOT reach the client distinctly on ANY of the eight rows in
// this feature. Only "already taken/in use" (name_taken) has a precise wire
// arm anywhere in the family — on this op specifically. The other three
// reasons are indistinguishable from a generic server error to every GMS/JMS
// client build examined (v48 through v95; JMS lacks the feature outright).
// A caller that needs the player to see "that name is reserved" or "that name
// has invalid characters" rendered as such cannot do so through any packet in
// this family — the client build has no string for it. That is a fact about
// the client binaries, not a shortcut taken in this codec.
//
// Derivation: docs/tasks/task-227-cash-name-change-world-transfer/derivation.md.
// packet-audit:fname CCashShop::OnCheckDuplicatedIDResult
type CheckNameChange struct {
	name   string
	result int8
}

func NewCheckNameChange(name string, result int8) CheckNameChange {
	return CheckNameChange{name: name, result: result}
}

// Name is the character name the client is checking. Echoed back to the
// client, which stashes it into the rename dialog on the available arm.
func (m CheckNameChange) Name() string { return m.name }

// Result is the resolved wire byte, NOT a semantic code, reinterpreted as
// signed to match the client's comparison. It comes from the tenant
// template's operations table via one of the body builders below.
func (m CheckNameChange) Result() int8 { return m.result }

func (m CheckNameChange) Operation() string {
	return CashShopCheckNameChangeWriter
}

func (m CheckNameChange) String() string {
	return fmt.Sprintf("name [%s], result [%d]", m.name, m.result)
}

func (m CheckNameChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		w.WriteInt8(m.result)
		return w.Bytes()
	}
}

func (m *CheckNameChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		m.result = r.ReadInt8()
	}
}

// CheckNameChangeAvailableBody emits the nResult == 0 arm. name is echoed
// back to the client's rename dialog.
func CheckNameChangeAvailableBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CheckNameChangeAvailable, func(code byte) packet.Encoder {
		return NewCheckNameChange(name, int8(code))
	})
}

// CheckNameChangeRejectedBody emits a refusal arm for a server-side rejection
// reason from the closed taxonomy in design.md §6. See the type doc comment
// ("This row owns the name-validity reason taxonomy — and cannot fully carry
// it") for why only name_taken maps to a distinct arm.
func CheckNameChangeRejectedBody(name string, reason string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return CheckNameChangeResultBody(name, checkNameChangeReasonKey(reason))
}

// checkNameChangeReasonKey maps a design.md §6 server-side reason onto the
// operations-table key whose configured byte this client version renders.
func checkNameChangeReasonKey(reason string) string {
	if key, ok := checkNameChangeReasonArms[reason]; ok {
		return key
	}
	return CheckNameChangeUnknownError
}

// checkNameChangeReasonArms is the reason -> arm table as DATA, so a test can
// assert it covers the four reasons this row must carry rather than trusting
// a switch to have listed them.
var checkNameChangeReasonArms = map[string]string{
	"name_taken":           CheckNameChangeTaken,
	"name_reserved":        CheckNameChangeUnknownError,
	"name_invalid_length":  CheckNameChangeUnknownError,
	"name_invalid_charset": CheckNameChangeUnknownError,
}

// CheckNameChangeResultBody emits an arbitrary arm by its operations-table
// key — the general form the two builders above specialise. key must be one
// of the CheckNameChange* constants; anything else resolves to ResolveCode's
// 99 sentinel, which as an int8 (99) is positive and lands in the client's
// TAKEN arm — not UNKNOWN_ERROR — so callers should route through
// CheckNameChangeRejectedBody rather than an unrecognised key here.
func CheckNameChangeResultBody(name string, key string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", key, func(code byte) packet.Encoder {
		return NewCheckNameChange(name, int8(code))
	})
}
