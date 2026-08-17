package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CashShopCheckNameChangePossibleHandle = "CashShopCheckNameChangePossibleHandle"

// CheckNameChangePossible is the serverbound NAME_TRANSFER op: the client's
// "may this character be renamed?" request, sent from the cash shop when the
// player buys the 5400000 name-change item.
//
// It is NOT a CASHSHOP_OPERATION mode arm — it gets its own opcode with no
// leading mode byte, and the body starts immediately with the character id.
// Opcode per version (derivation.md §1.2, confirmed against the COutPacket
// ctor in each IDB):
//
//	gms_v48 0x012  — NOTE: v48 predates the two-slot shift; the v61+ constant
//	                 is WRONG here (COutPacket::COutPacket(v5, 18) @0x44f955).
//	gms_v61 0x010, gms_v72 0x010, gms_v79 0x010, gms_v83 0x010, gms_v84 0x010,
//	gms_v87 0x010, gms_v92 0x010, gms_v95 0x010
//	jms_v185  — ABSENT. That client has no name-change feature at all: no
//	            5400000 arm in CCashShop::ProcessBuy @0x484ca8, zero hits for
//	            the immediate 5400000 in the whole binary, and no
//	            CUIChangingCharacterName / OnCheckNameChangePossibleResult /
//	            OnCheckDuplicatedIDResult. Its serverbound 0x009 is
//	            WORLD_TRANSFER, which an earlier transposed IDB symbol had
//	            filed under this op (derivation.md §1.5).
//
// Body (derivation.md §2.1):
//
//	gms_v48 … gms_v92 : Encode4 dwCharacterID · Encode4 nBirthDate
//	gms_v95           : Encode4 dwCharID      · EncodeStr sSPW
//
// The second field CHANGES WIRE TYPE at v95 — it is not a widening. Pre-v95
// the credential is an 8-digit integer birthday code: v83 ask_SPW @0x9acf95
// ends `v8 = atoi(*InputStr_Result); … return v8;` (and returns -1 on cancel),
// prompting SP_826 "…please enter your 8 digit birthday code…". The
// `ZXString<char>` in the pre-v95 mangled prototypes is wrong. gms_v95 is
// PDB-backed and names the parameter sSPW, put on the wire by
// COutPacket::EncodeStr @0x4881f7.
//
// This op is NOT swapped with WORLD_TRANSFER. The disambiguator is
// CCashShop::ProcessBuy, which compares the cash item id against the exact
// constants 5400000 and 5401000 and routes each to its own OnBuy* arm; only
// the 5401000 → COutPacket(0x12) arm is gated by
// CCashShop::CheckTransferWorldPossible (the world-transfer refusals). See
// derivation.md §1.1 — a swapped decoder is structurally plausible here and
// still passes its own byte fixture, so the symbol names are not evidence.
//
// Derivation: docs/tasks/task-227-cash-name-change-world-transfer/derivation.md.
// packet-audit:fname CCashShop::SendCheckNameChangePossiblePacket
type CheckNameChangePossible struct {
	characterId uint32
	birthDate   uint32
	spw         string
}

func NewCheckNameChangePossible(characterId uint32, birthDate uint32, spw string) CheckNameChangePossible {
	return CheckNameChangePossible{characterId: characterId, birthDate: birthDate, spw: spw}
}

func (m CheckNameChangePossible) CharacterId() uint32 { return m.characterId }

// BirthDate is the pre-v95 form of the account credential: the 8-digit
// birthday code the client collects through ask_SPW and puts on the wire as a
// uint32. Zero on gms_v95, which sends the credential as Spw instead.
func (m CheckNameChangePossible) BirthDate() uint32 { return m.birthDate }

// Spw is the gms_v95 form of the same credential — the account second
// password, length-prefixed on the wire. Empty on gms_v48…gms_v92, which send
// BirthDate instead.
func (m CheckNameChangePossible) Spw() string { return m.spw }

func (m CheckNameChangePossible) Operation() string {
	return CashShopCheckNameChangePossibleHandle
}

// String REDACTS the credential. Both birthDate and spw carry the account's
// second password / birthday code, and every serverbound handler in
// atlas-channel logs p.String() at debug level — so redaction has to live here,
// in the struct, not at each call site. A credential in a log line is a
// credential anyone with log access owns. Only the character id, which is not a
// secret and is the field an operator needs when triaging a rename request, is
// logged verbatim.
func (m CheckNameChangePossible) String() string {
	return fmt.Sprintf("characterId [%d], credential [REDACTED]", m.characterId)
}

// CredentialIsString reports whether this version puts the credential on the
// wire as a length-prefixed string (EncodeStr sSPW) rather than as a uint32
// birthday code (Encode4 nBirthDate). Derived per version: gms_v48 through
// gms_v92 encode the integer (two Encode4 sites in the send path);
// gms_v95 encodes the string (Encode4 + EncodeStr @0x4881f7).
//
// The gate is MajorAtLeast(95), never a raw `> 92`/`> 83` comparison: the
// boundary is v95, and v84 must take the same path as v83.
//
// jms_v185 has no NAME_TRANSFER at all (see the type doc), so no jms cell is
// claimed and no jms fixture exists. It falls on the string side here only
// because its major version is 185; the jms behaviour of this codec is
// unverified and unused.
//
// Exported so atlas-channel's handler can drive the SAME predicate that
// decoded the body when deciding which stored account credential
// (account.Model.PIC() vs account.Model.BirthDate()) to compare against — a
// second, independently-written gate in the handler could drift from this one
// and validate a field the decoder never populated (task-227 Task 26 ruling
// 2).
func CredentialIsString(ctx context.Context) bool {
	return credentialIsString(ctx)
}

// credentialIsString is the unexported implementation Encode/Decode call
// directly; CredentialIsString is its exported alias for external callers.
func credentialIsString(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.MajorAtLeast(95)
}

func (m CheckNameChangePossible) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		if credentialIsString(ctx) {
			w.WriteAsciiString(m.spw)
		} else {
			w.WriteInt(m.birthDate)
		}
		return w.Bytes()
	}
}

func (m *CheckNameChangePossible) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		if credentialIsString(ctx) {
			m.spw = r.ReadAsciiString()
		} else {
			m.birthDate = r.ReadUint32()
		}
	}
}
