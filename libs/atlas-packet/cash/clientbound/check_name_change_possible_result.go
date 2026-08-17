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

const CashShopCheckNameChangePossibleResultWriter = "CashShopCheckNameChangePossibleResult"

// Reason-code keys for the tenant socket-config `operations` table of the
// CashShopCheckNameChangePossibleResult writer. The VALUES live in the six
// applicable seed templates, never here (DOM-25) — a key that is missing from a
// template resolves to ResolveCode's loud 99 sentinel, which lands in the
// client's default arm and renders the unknown-error notice.
//
// The enumeration is the client's own, read out of
// CCashShop::OnCheckNameChangePossibleResult on every applicable version. The
// handler reads nResult and branches four ways; nResult never changes the field
// layout, only which dialog or notice renders — which is why this op is a flat
// codec and not a dispatcher family:
//
//	0        open CUIChangingLicenseNotice + SetBirthDate(nBirthDate)
//	1        Notice — "the name change is already submitted due to the item purchase"
//	2        Notice — request limit, "…please check if you were recently…"
//	3        Notice — request limit, "…please check if you requested…"
//	default  Notice — "an unknown error has occurred"
//
// StringPool ids per version for arms 1 / 2 / 3 / default
// (derivation.md §4.1, re-read from each IDB this pass):
//
//	gms_v79  523 / 524 / 525 / 3550
//	gms_v83  521 / 522 / 523 / 3570   (enum members carry the text verbatim,
//	                                   e.g. SP_521_THE_NAME_CHANGE_IS_ALREADY_SUBMITTED__R_NDUE_TO_THE_ITEM_PURCHASE)
//	gms_v84  524 / 525 / 526 / 3573
//	gms_v87  531 / 532 / 533 / 3579
//	gms_v92  538 / 539 / 540 / 3639
//	gms_v95  538 / 539 / 540 / 3606
//
// The rendered strings for the numeric-only versions are NOT recorded: the
// StringPoolStrings enum is not member-enumerable in those IDBs. The numeric id
// is what the wire contract needs; the text is client-side presentation.
const (
	// CheckNameChangePossibleAllowed is arm 0 — the rename may proceed. The
	// client opens CUIChangingLicenseNotice and feeds it the birth date.
	CheckNameChangePossibleAllowed = "ALLOWED"
	// CheckNameChangePossibleAlreadySubmitted is arm 1.
	CheckNameChangePossibleAlreadySubmitted = "ALREADY_SUBMITTED"
	// CheckNameChangePossibleRequestLimitRecent is arm 2.
	CheckNameChangePossibleRequestLimitRecent = "REQUEST_LIMIT_RECENT"
	// CheckNameChangePossibleRequestLimitRequested is arm 3.
	CheckNameChangePossibleRequestLimitRequested = "REQUEST_LIMIT_REQUESTED"
	// CheckNameChangePossibleUnknownError is the client's `default:` arm —
	// every byte that is not 0, 1, 2 or 3. gms_v95 @0x4954c8 compiles it as a
	// literal `switch (v3) { case 0: … case 3: … default: … }` and gms_v79…v92
	// as the equivalent decrement chain, so any value >= 4 provably lands
	// here. The templates configure 4 as the canonical representative; it is
	// config-resolved so an operator can move it without a code change.
	CheckNameChangePossibleUnknownError = "UNKNOWN_ERROR"
)

// CheckNameChangePossibleResult is the clientbound
// CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT op — the server's answer to the
// cash-shop NAME_TRANSFER request (cash/serverbound.CheckNameChangePossible),
// received by CCashShop::OnCheckNameChangePossibleResult.
//
// Body, IDENTICAL on every applicable version — v79, v83, v84, v87, v92, v95
// (derivation.md §2.4, independently re-decompiled this pass):
//
//	Decode4  dwCharacterID   // read and DISCARDED by the client
//	Decode1  nResult
//	Decode4  nBirthDate      // gms_v95's PDB names this local nBirthDate
//
// There is deliberately NO version gate in Encode/Decode: no field, width or
// order differs across the six versions. The only per-version value is the
// opcode, which lives in the tenant template. Receiver addresses: v79 0x474ab1,
// v83 0x47bbb6, v84 0x47ed54, v87 0x487397, v92 0x4913a0, v95 0x495470.
//
// dwCharacterID is on the wire but the client throws it away — every version's
// first statement is a bare `CInPacket::Decode4(iPacket);` with no assignment.
// It is modelled anyway because the field occupies four bytes that the rest of
// the body is positioned behind; dropping it would desync nResult.
//
// nBirthDate is read unconditionally, BEFORE the result switch, so it is on the
// wire on failure paths too — only arm 0 consumes it
// (CUIChangingLicenseNotice::SetBirthDate, gms_v95 @0x495523).
//
// Absent from gms_v48/v61/v72 and jms_v185: those clients have no
// OnCheckNameChangePossibleResult receiver at all (the op arrives with the
// name-change licence dialog at v79), and jms_v185 has no name-change feature
// whatsoever (derivation.md §1.5).
//
// Derivation: docs/tasks/task-227-cash-name-change-world-transfer/derivation.md.
// packet-audit:fname CCashShop::OnCheckNameChangePossibleResult
type CheckNameChangePossibleResult struct {
	characterId uint32
	result      byte
	birthDate   uint32
}

func NewCheckNameChangePossibleResult(characterId uint32, result byte, birthDate uint32) CheckNameChangePossibleResult {
	return CheckNameChangePossibleResult{characterId: characterId, result: result, birthDate: birthDate}
}

func (m CheckNameChangePossibleResult) CharacterId() uint32 { return m.characterId }

// Result is the resolved wire byte, NOT a semantic code. It comes from the
// tenant template's operations table via one of the body builders below.
func (m CheckNameChangePossibleResult) Result() byte { return m.result }

// BirthDate is the 8-digit birthday code the client hands to
// CUIChangingLicenseNotice on the success arm. Unlike the serverbound
// credential it is echoed back by the server rather than supplied by the
// player, and the client renders it in a dialog, so it is not redacted here.
func (m CheckNameChangePossibleResult) BirthDate() uint32 { return m.birthDate }

func (m CheckNameChangePossibleResult) Operation() string {
	return CashShopCheckNameChangePossibleResultWriter
}

func (m CheckNameChangePossibleResult) String() string {
	return fmt.Sprintf("characterId [%d], result [%d], birthDate [%d]", m.characterId, m.result, m.birthDate)
}

func (m CheckNameChangePossibleResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.result)
		w.WriteInt(m.birthDate)
		return w.Bytes()
	}
}

func (m *CheckNameChangePossibleResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.result = r.ReadByte()
		m.birthDate = r.ReadUint32()
	}
}

// CheckNameChangePossibleResultAllowedBody emits the success arm. birthDate is
// echoed to the client, which feeds it to CUIChangingLicenseNotice.
func CheckNameChangePossibleResultAllowedBody(characterId uint32, birthDate uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CheckNameChangePossibleAllowed, func(code byte) packet.Encoder {
		return NewCheckNameChangePossibleResult(characterId, code, birthDate)
	})
}

// CheckNameChangePossibleResultRejectedBody emits a refusal arm for a
// server-side rejection reason from the closed taxonomy in
// design.md §6. The mapping, reason → operations-table key → client arm:
//
//	name_taken           -> UNKNOWN_ERROR   (client default arm)
//	name_reserved        -> UNKNOWN_ERROR   (client default arm)
//	name_invalid_length  -> UNKNOWN_ERROR   (client default arm)
//	name_invalid_charset -> UNKNOWN_ERROR   (client default arm)
//
// All four land on the same arm, and that is a fact about the client, not a
// shortcut. This packet's enumeration has exactly four arms and none of them
// renders a name-validity message: arm 1 says the rename was ALREADY SUBMITTED
// for this purchase, arms 2 and 3 say the account hit a REQUEST LIMIT. There is
// no "that name is taken" / "that name is reserved" / "that name is too short"
// / "that name has illegal characters" string in
// CCashShop::OnCheckNameChangePossibleResult on any of the six versions —
// gms_v83 @0x47bbb6 references exactly SP_521, SP_522, SP_523 and SP_3570, and
// the other five are structurally identical (derivation.md §4.1).
//
// Name validity travels on a DIFFERENT op. CASHSHOP_CHECK_NAME_CHANGE
// (CCashShop::OnCheckDuplicatedIDResult, derivation.md §2.3) carries
// DecodeStr sName + Decode1 nResult, where nResult > 0 is "already in use" and
// == 0 is "available" — that is where the client renders a name-specific
// answer, and that is where a name_taken / name_reserved rejection belongs. So
// routing the four validity reasons to this op's unknown-error arm loses no
// information the client could have displayed; inventing a distinct wire code
// per reason would instead put bytes on the wire that every version's switch
// falls through to default anyway, while implying a client behaviour that does
// not exist.
//
// The two reasons this op CAN express precisely — already_pending (arm 1) and a
// rate limit (arms 2/3) — are emitted by name through
// CheckNameChangePossibleResultBody rather than through this reason mapper,
// because they are not part of the four-reason set this row is required to
// cover.
//
// An unrecognised reason string also resolves UNKNOWN_ERROR: the taxonomy is
// closed, so a value outside it is a server bug, and the safe wire answer is
// the arm the client already treats as "something went wrong".
func CheckNameChangePossibleResultRejectedBody(characterId uint32, reason string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return CheckNameChangePossibleResultBody(characterId, checkNameChangePossibleReasonKey(reason), 0)
}

// checkNameChangePossibleReasonKey maps a design.md §6 server-side reason onto
// the operations-table key whose configured byte this client version renders.
// See CheckNameChangePossibleResultRejectedBody for why all four members of
// this row's required set share one arm.
func checkNameChangePossibleReasonKey(reason string) string {
	if key, ok := checkNameChangePossibleReasonArms[reason]; ok {
		return key
	}
	return CheckNameChangePossibleUnknownError
}

// checkNameChangePossibleReasonArms is the reason → arm table as DATA, so a
// test can assert it covers the four reasons this row must carry rather than
// trusting a switch to have listed them.
var checkNameChangePossibleReasonArms = map[string]string{
	"name_taken":           CheckNameChangePossibleUnknownError,
	"name_reserved":        CheckNameChangePossibleUnknownError,
	"name_invalid_length":  CheckNameChangePossibleUnknownError,
	"name_invalid_charset": CheckNameChangePossibleUnknownError,
}

// CheckNameChangePossibleResultBody emits an arbitrary arm by its
// operations-table key — the general form the two builders above specialise.
// key must be one of the CheckNameChangePossible* constants; anything else
// resolves to ResolveCode's 99 sentinel, which the client renders as the
// unknown-error notice.
func CheckNameChangePossibleResultBody(characterId uint32, key string, birthDate uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", key, func(code byte) packet.Encoder {
		return NewCheckNameChangePossibleResult(characterId, code, birthDate)
	})
}
