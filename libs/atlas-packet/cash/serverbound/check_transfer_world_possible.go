package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CashShopCheckTransferWorldPossibleHandle = "CashShopCheckTransferWorldPossibleHandle"

// CheckTransferWorldPossible is the serverbound WORLD_TRANSFER op: the client's
// "may this character change worlds?" request, sent from the cash shop when the
// player buys the 5401000 world-transfer item and the client-side gate
// CCashShop::CheckTransferWorldPossible passes.
//
// It is NOT a CASHSHOP_OPERATION mode arm — it gets its own opcode with no
// leading mode byte. Opcode per version (derivation.md §1.2, each confirmed
// against the COutPacket ctor in that version's own IDB during this task):
//
//	gms_v48   0x014  — NOTE: v48 predates the two-slot shift; the v61+ constant
//	                   is WRONG here (COutPacket::COutPacket(v5, 20) @0x44fbac).
//	gms_v61 0x012, gms_v72 0x012, gms_v79 0x012, gms_v83 0x012, gms_v84 0x012,
//	gms_v87 0x012, gms_v92 0x012, gms_v95 0x012
//	jms_v185  0x009  (COutPacket::COutPacket(v6, 9) @0x484fd8)
//
// Body (derivation.md §2.2):
//
//	gms_v48 … gms_v92 : Encode4 dwCharacterID · Encode4 nBirthDate
//	gms_v95           : Encode4 dwCharacterID · EncodeStr sSPW
//	jms_v185          : EncodeStr sSPW ONLY — no character id at all
//
// Two independent divergences, both gated below rather than assumed:
//
//  1. The credential CHANGES WIRE TYPE at gms_v95 — it is not a widening.
//     Pre-v95 it is an 8-digit integer birthday code: v83 ask_SPW @0x9acf95
//     ends `v8 = atoi(*InputStr_Result); … return v8;` and returns -1 on
//     cancel, prompting SP_826 "…please enter your 8 digit birthday code…".
//     The `ZXString<char>` in the pre-v95 mangled prototypes is wrong.
//     gms_v95 is PDB-backed and names the parameter sSPW, put on the wire by
//     COutPacket::EncodeStr @0x488527.
//  2. jms_v185 drops the character id entirely — a structurally different body,
//     not a version tweak of the GMS one. The applied two-argument type on
//     that IDB is wrong: `retn 4` @0x485035 proves a single stack argument,
//     and the send @0x484fbf performs exactly one EncodeStr @0x484ff6 with no
//     Encode4 anywhere in the function.
//
// This op is NOT swapped with NAME_TRANSFER, despite the transposed IDB
// symbols in this neighbourhood (derivation.md §1.4 lists the corrections).
// The disambiguator is CCashShop::ProcessBuy, which compares the cash item id
// against the exact constants 5400000 and 5401000 and routes each to its own
// OnBuy* arm; only the 5401000 arm — the one that reaches this send — is gated
// by CCashShop::CheckTransferWorldPossible, the function that formats the
// world-transfer refusals ("Guild Master can not transfer worlds.", "GM can
// not transfer worlds.", SP_5017 "you have to quit family to move to another
// world"). See derivation.md §1.1: a swapped decoder here is structurally
// plausible and would still pass its own byte fixture, so the symbol names are
// not evidence.
//
// Derivation: docs/tasks/task-227-cash-name-change-world-transfer/derivation.md.
// packet-audit:fname CCashShop::SendCheckTransferWorldPossiblePacket
type CheckTransferWorldPossible struct {
	characterId uint32
	birthDate   uint32
	spw         string
}

func NewCheckTransferWorldPossible(characterId uint32, birthDate uint32, spw string) CheckTransferWorldPossible {
	return CheckTransferWorldPossible{characterId: characterId, birthDate: birthDate, spw: spw}
}

// CharacterId is the character the player asked to transfer. Zero on jms_v185,
// whose body carries no character id — that client identifies the character
// from session state alone.
func (m CheckTransferWorldPossible) CharacterId() uint32 { return m.characterId }

// BirthDate is the pre-v95 form of the account credential: the 8-digit
// birthday code the client collects through ask_SPW and puts on the wire as a
// uint32. Zero on gms_v95 and jms_v185, which send the credential as Spw.
func (m CheckTransferWorldPossible) BirthDate() uint32 { return m.birthDate }

// Spw is the gms_v95 / jms_v185 form of the same credential — the account
// second password, length-prefixed on the wire. Empty on gms_v48…gms_v92,
// which send BirthDate instead.
func (m CheckTransferWorldPossible) Spw() string { return m.spw }

func (m CheckTransferWorldPossible) Operation() string {
	return CashShopCheckTransferWorldPossibleHandle
}

// String REDACTS the credential. Both birthDate and spw carry the account's
// second password / birthday code, and every serverbound handler in
// atlas-channel logs p.String() at debug level — so redaction has to live here,
// in the struct, not at each call site. A credential in a log line is a
// credential anyone with log access owns. Only the character id, which is not a
// secret and is the field an operator needs when triaging a transfer request,
// is logged verbatim.
func (m CheckTransferWorldPossible) String() string {
	return fmt.Sprintf("characterId [%d], credential [REDACTED]", m.characterId)
}

// TransferBodyHasCharacterId is the exported alias of transferBodyHasCharacterId
// for external callers (task-227 Task 26 ruling 2) — see that function's doc
// comment for the derivation.
func TransferBodyHasCharacterId(ctx context.Context) bool {
	return transferBodyHasCharacterId(ctx)
}

// transferBodyHasCharacterId reports whether this version prefixes the body
// with Encode4 dwCharacterID. Every GMS version v48…v95 does. jms_v185 does
// NOT: its send is a single EncodeStr, proven by `retn 4` @0x485035 (one stack
// argument under __thiscall) and by the absence of any Encode4 call in
// CCashShop::SendCheckTransferWorldPossiblePacket @0x484fbf.
//
// The gate is on region, not on version: this is a structural difference
// between the two client families, not a GMS timeline boundary, so no
// MajorAtLeast threshold can express it. JMS is the only region derived as
// id-less; every other region keeps the GMS shape.
func transferBodyHasCharacterId(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.Region() != "JMS"
}

// transferCredentialIsString reports whether this version puts the credential
// on the wire as a length-prefixed string (EncodeStr sSPW) rather than as a
// uint32 birthday code (Encode4 nBirthDate). Derived per version: gms_v48
// through gms_v92 encode the integer (two Encode4 sites in the send path);
// gms_v95 encodes the string (Encode4 @0x488507 + EncodeStr @0x488527); and
// jms_v185 encodes the string (EncodeStr @0x484ff6).
//
// The GMS gate is MajorAtLeast(95), never a raw `> 92`/`> 83` comparison: the
// boundary is v95, and v84 must take the same path as v83. The JMS arm is
// derived independently rather than left to fall out of MajorAtLeast(185 ≥ 95)
// — the two happen to agree today, but jms_v185's string credential is a fact
// about that client's send site, not about a GMS version threshold.
func transferCredentialIsString(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	if t.Region() == "JMS" {
		return true
	}
	return t.MajorAtLeast(95)
}

// TransferCredentialIsString is the exported alias of
// transferCredentialIsString for external callers (task-227 Task 26 ruling
// 2) — atlas-channel's handler drives this SAME predicate rather than
// re-deriving a MajorAtLeast(95) check that would miss the JMS arm. See
// transferCredentialIsString's doc comment for the derivation.
func TransferCredentialIsString(ctx context.Context) bool {
	return transferCredentialIsString(ctx)
}

func (m CheckTransferWorldPossible) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if transferBodyHasCharacterId(ctx) {
			w.WriteInt(m.characterId)
		}
		if transferCredentialIsString(ctx) {
			w.WriteAsciiString(m.spw)
		} else {
			w.WriteInt(m.birthDate)
		}
		return w.Bytes()
	}
}

func (m *CheckTransferWorldPossible) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		if transferBodyHasCharacterId(ctx) {
			m.characterId = r.ReadUint32()
		}
		if transferCredentialIsString(ctx) {
			m.spw = r.ReadAsciiString()
		} else {
			m.birthDate = r.ReadUint32()
		}
	}
}
