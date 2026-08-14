package clientbound

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CashShopCheckTransferWorldPossibleResultWriter = "CashShopCheckTransferWorldPossibleResult"

// Reason-code keys for the tenant socket-config `operations` table of the
// CashShopCheckTransferWorldPossibleResult writer. The VALUES live in the ten
// applicable seed templates, never here (DOM-25).
//
// The enumeration is CCashShop::OnCheckTransferWorldPossibleResult's own
// switch on nResult, independently re-decompiled on v83 this pass
// (derivation.md §2.5/§4.2 confirmed by direct decompile of v83 @0x47bd9b):
//
//	0        open CUITransferWorldLicenseNotice (+SetBirthDate except jms)
//	1        Notice — literal "Cannot find Character Information." (NOT StringPool)
//	2..7     Notice(SP) — StringPool ids that differ per version; no per-version
//	         text is recoverable (the StringPoolStrings enum has no member names
//	         for these ids in any of the ten IDBs — type_inspect member_count: 0)
//	8        Notice(SP_5017) — "you have to quit family to move to another world"
//	default  Notice(SP) — "an unknown error has occurred"
//
// Arm 8's text is the ONE numeric id in 2..8 with independently confirmed
// content: v83 @0x47bd9b loads the literal 5017 into the StringPool lookup,
// and CCashShop::CheckTransferWorldPossible @0x4734e5 — the CLIENT-SIDE
// pre-send gate, a different function entirely — references the SAME id by
// its named enum constant SP_5017_YOU_HAVE_TO_QUIT_FAMILY__R_NTO_MOVE_TO_ANOTHER_WORLD.
// That cross-reference is the only place any of these ids carries decoded text.
//
// StringPool ids per version for arms 1(literal)/2/3/4/5/6/7/8/default
// (derivation.md §4.2, table columns v48..jms; v48/v61 have NO case 8 — their
// switch ends at 7 and any nResult==8 falls to default):
//
//	gms_v48   *(unverified)/3655/3661/3656/3657/3667/3662/n-a /3222
//	gms_v61   lit/3927/3933/3928/3929/3939/3934/n-a /3492
//	gms_v72   lit/3978/3985/3979/3980/3991/3986/4996/3546
//	gms_v79   lit/3981/3988/3982/3983/3994/3989/4979/3550
//	gms_v83   lit/4002/4009/4003/4004/4015/4010/5017/3570
//	gms_v84   lit/4005/4012/4006/4007/4018/4013/5022/3573
//	gms_v87   lit/4010/4017/4011/4012/4023/4018/5028/3579
//	gms_v92   lit/4076/4083/4077/4078/4089/4084/5095/3639
//	gms_v95   lit/4043/4050/4044/4045/4056/4051/5035/3606
//	jms_v185  SP5459/5452/5458/5453/5454/5465/5460/5559/5482
//
// jms_v185's arm 1 is StringPool-backed (not the GMS literal), and every id in
// its column shifts, but the SHAPE of the switch (9 numbered arms + default)
// is the same as GMS.
const (
	// CheckTransferWorldPossibleAllowed is arm 0 — the transfer may proceed.
	CheckTransferWorldPossibleAllowed = "ALLOWED"
	// CheckTransferWorldPossibleCharacterNotFound is arm 1 — literal
	// "Cannot find Character Information." on GMS, StringPool-backed on jms.
	// No design-taxonomy reason is faithfully this on any version (see
	// checkTransferWorldPossibleReasonArms below); the constant is exposed
	// because it is a real, distinct client arm.
	CheckTransferWorldPossibleCharacterNotFound = "CHARACTER_NOT_FOUND"
	// CheckTransferWorldPossibleReason2..Reason7 are arms 2..7. None of the ten
	// IDBs has a decoded StringPool text for these ids (§4.2) — the constants
	// exist so the wire's full arm space is representable, but the reason
	// mapper below does not route any design-taxonomy reason onto them: doing
	// so would assert a semantic meaning ("world full", "banned", …) that no
	// decompiled string confirms.
	CheckTransferWorldPossibleReason2 = "REASON_2"
	CheckTransferWorldPossibleReason3 = "REASON_3"
	CheckTransferWorldPossibleReason4 = "REASON_4"
	CheckTransferWorldPossibleReason5 = "REASON_5"
	CheckTransferWorldPossibleReason6 = "REASON_6"
	CheckTransferWorldPossibleReason7 = "REASON_7"
	// CheckTransferWorldPossibleInFamily is arm 8 — the one rejection arm with
	// independently confirmed text (see the doc comment above). Absent on
	// gms_v48/gms_v61 (their switch has no case 8; a resolved code of 8 falls
	// to those two versions' `default:` arm, which is harmless — the reason
	// still renders as an error notice, just the generic one).
	CheckTransferWorldPossibleInFamily = "IN_FAMILY"
	// CheckTransferWorldPossibleUnknownError is the client's `default:` arm.
	CheckTransferWorldPossibleUnknownError = "UNKNOWN_ERROR"
)

// CheckTransferWorldPossibleResult is the clientbound
// CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT op — the server's answer to
// the cash-shop WORLD_TRANSFER request (cash/serverbound.CheckTransferWorldPossible),
// received by CCashShop::OnCheckTransferWorldPossibleResult.
//
// Body, GMS v48..v95 (derivation.md §2.5, re-confirmed by direct decompile of
// v83 @0x47bd9b this pass):
//
//	Decode4  dwCharacterID   // read and DISCARDED by the client
//	Decode1  nResult
//	Decode4  nBirthDate      // read UNCONDITIONALLY, before the result switch —
//	                         // on the wire on every arm, not just arm 0
//	Decode1  bHasWorldList
//	  if bHasWorldList:
//	    Decode4  nCount
//	    nCount x DecodeStr    // -> CCashShop::m_asWorldName
//
// jms_v185 (@0x48e7a6) DROPS nBirthDate — its success arm constructs
// CUITransferWorldLicenseNotice with no SetBirthDate call:
//
//	Decode4  dwCharacterID
//	Decode1  nResult
//	Decode1  bHasWorldList
//	  if bHasWorldList: Decode4 nCount; nCount x DecodeStr
//
// The gate is on REGION, not on a version threshold — jms_v185 is structurally
// different from every GMS version, not a GMS timeline boundary — so it uses a
// t.Region() check, never MajorAtLeast/raw comparison (mirrors the identical
// gate on the sibling serverbound op, cash/serverbound.CheckTransferWorldPossible).
//
// The world-name list is decoded BEFORE the result switch on every version, so
// it is present on failure paths too, not just arm 0.
//
// Receiver addresses (derivation.md §2.5): v48 0x455d25, v61 0x463ba6,
// v72 0x4737ca, v79 0x474c96, v83 0x47bd9b, v84 0x47ef39, v87 0x48757c,
// v92 0x494040, v95 0x4980b0, jms_v185 0x48e7a6.
//
// Derivation: docs/tasks/task-227-cash-name-change-world-transfer/derivation.md.
// packet-audit:fname CCashShop::OnCheckTransferWorldPossibleResult
type CheckTransferWorldPossibleResult struct {
	characterId  uint32
	result       byte
	birthDate    uint32
	hasWorldList bool
	worldNames   []string
}

func NewCheckTransferWorldPossibleResult(characterId uint32, result byte, birthDate uint32, worldNames []string) CheckTransferWorldPossibleResult {
	return CheckTransferWorldPossibleResult{
		characterId:  characterId,
		result:       result,
		birthDate:    birthDate,
		hasWorldList: len(worldNames) > 0,
		worldNames:   worldNames,
	}
}

func (m CheckTransferWorldPossibleResult) CharacterId() uint32 { return m.characterId }

// Result is the resolved wire byte, NOT a semantic code. It comes from the
// tenant template's operations table via one of the body builders below.
func (m CheckTransferWorldPossibleResult) Result() byte { return m.result }

// BirthDate is the 8-digit birthday code echoed back to
// CUITransferWorldLicenseNotice::SetBirthDate on the success arm. Always zero
// on jms_v185 — that region's body carries no such field at all (not merely a
// zero value on the wire; the byte is not encoded/decoded on jms).
func (m CheckTransferWorldPossibleResult) BirthDate() uint32 { return m.birthDate }

// HasWorldList is the bHasWorldList wire flag.
func (m CheckTransferWorldPossibleResult) HasWorldList() bool { return m.hasWorldList }

// WorldNames is the decoded m_asWorldName list. Empty when HasWorldList is
// false.
func (m CheckTransferWorldPossibleResult) WorldNames() []string { return m.worldNames }

func (m CheckTransferWorldPossibleResult) Operation() string {
	return CashShopCheckTransferWorldPossibleResultWriter
}

func (m CheckTransferWorldPossibleResult) String() string {
	return fmt.Sprintf("characterId [%d], result [%d], birthDate [%d], worldNames [%s]",
		m.characterId, m.result, m.birthDate, strings.Join(m.worldNames, ","))
}

// checkTransferWorldPossibleResultHasBirthDate reports whether this region's
// body carries the unconditional nBirthDate field. False only for JMS — see
// the type doc comment for the structural (not version-threshold) reasoning.
func checkTransferWorldPossibleResultHasBirthDate(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.Region() != "JMS"
}

func (m CheckTransferWorldPossibleResult) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.result)
		if checkTransferWorldPossibleResultHasBirthDate(ctx) {
			w.WriteInt(m.birthDate)
		}
		w.WriteBool(m.hasWorldList)
		if m.hasWorldList {
			w.WriteInt(uint32(len(m.worldNames)))
			for _, name := range m.worldNames {
				w.WriteAsciiString(name)
			}
		}
		return w.Bytes()
	}
}

func (m *CheckTransferWorldPossibleResult) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.result = r.ReadByte()
		if checkTransferWorldPossibleResultHasBirthDate(ctx) {
			m.birthDate = r.ReadUint32()
		}
		m.hasWorldList = r.ReadBool()
		if m.hasWorldList {
			count := int(r.ReadUint32())
			m.worldNames = make([]string, count)
			for i := 0; i < count; i++ {
				m.worldNames[i] = r.ReadAsciiString()
			}
		}
	}
}

// CheckTransferWorldPossibleResultAllowedBody emits the success arm.
// birthDate is ignored (encoded as zero) on regions whose body has no such
// field (see checkTransferWorldPossibleResultHasBirthDate).
func CheckTransferWorldPossibleResultAllowedBody(characterId uint32, birthDate uint32, worldNames []string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CheckTransferWorldPossibleAllowed, func(code byte) packet.Encoder {
		return NewCheckTransferWorldPossibleResult(characterId, code, birthDate, worldNames)
	})
}

// CheckTransferWorldPossibleResultRejectedBody emits a refusal arm for a
// server-side rejection reason from the closed taxonomy the brief for this
// row enumerates. The mapping, reason -> operations-table key -> client arm:
//
//	in_family            -> IN_FAMILY        (arm 8, confirmed text — see the
//	                                           type-level doc comment)
//	world_same            -> UNKNOWN_ERROR    (client default arm)
//	world_unknown         -> UNKNOWN_ERROR    (client default arm)
//	world_full            -> UNKNOWN_ERROR    (client default arm)
//	no_character_slot     -> UNKNOWN_ERROR    (client default arm)
//	banned                -> UNKNOWN_ERROR    (client default arm)
//	is_guild_master       -> UNKNOWN_ERROR    (client default arm — see note)
//	is_gm                 -> UNKNOWN_ERROR    (client default arm — see note)
//	trade_open            -> UNKNOWN_ERROR    (client default arm)
//	merchant_open         -> UNKNOWN_ERROR    (client default arm)
//	mts_listings_open     -> UNKNOWN_ERROR    (client default arm)
//	name_taken            -> UNKNOWN_ERROR    (client default arm)
//
// Only in_family lands on a distinct arm, because arm 8 is the only rejection
// arm (of 1..8) with independently confirmed rendered text: v83
// @0x47bd9b's case 8 loads StringPool id 5017, and
// CCashShop::CheckTransferWorldPossible @0x4734e5 — a DIFFERENT function,
// the client-side pre-send eligibility gate — references that exact id by its
// named enum constant SP_5017_YOU_HAVE_TO_QUIT_FAMILY__R_NTO_MOVE_TO_ANOTHER_WORLD.
//
// is_guild_master and is_gm are deliberately NOT routed to a distinct arm even
// though CCashShop::CheckTransferWorldPossible (the same client-side gate
// function above) formats "Guild Master can not transfer worlds." and "GM can
// not transfer worlds." for exactly those conditions. That formatting happens
// BEFORE the client ever sends the WORLD_TRANSFER request — it is a local
// check inside OnBuyTransferWorld's call chain, gating whether
// SendCheckTransferWorldPossiblePacket is even called. It is not part of
// OnCheckTransferWorldPossibleResult's switch, and neither of those two exact
// strings, nor their ZXString<char>::Format literals, appears anywhere in the
// decompiled body of the result handler on any of the ten versions. A modified
// or race-condition client that reaches the server despite failing its own
// local guild-master/GM gate gets the generic UNKNOWN_ERROR notice, which is
// the safe answer — inventing a distinct wire code for a text that provably
// belongs to a different, unreachable-from-here function would assert a
// client behaviour that does not exist.
//
// The remaining nine reasons (world_same, world_unknown, world_full,
// no_character_slot, banned, trade_open, merchant_open, mts_listings_open,
// name_taken) collapse to UNKNOWN_ERROR for the same reason arms 2..7 are
// exposed but unused by this mapper: none of the ten IDBs has decoded
// StringPool text for ids 2..7 (§4.2 — the StringPoolStrings enum carries no
// member names for them), so assigning any of these nine business reasons to
// one of those six arms would be inventing a semantic pairing this task's
// evidence does not support. UNKNOWN_ERROR is the client's own designated
// catch-all for "something went wrong" and loses no information the client
// could have displayed more specifically.
//
// An unrecognised reason string also resolves UNKNOWN_ERROR: the taxonomy is
// closed, so a value outside it is a server bug, and the safe wire answer is
// the arm the client already treats as "something went wrong".
func CheckTransferWorldPossibleResultRejectedBody(characterId uint32, reason string, worldNames []string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return CheckTransferWorldPossibleResultBody(characterId, checkTransferWorldPossibleReasonKey(reason), 0, worldNames)
}

// checkTransferWorldPossibleReasonKey maps a server-side reason onto the
// operations-table key whose configured byte this client version renders. See
// CheckTransferWorldPossibleResultRejectedBody for why only in_family maps
// to a distinct arm.
func checkTransferWorldPossibleReasonKey(reason string) string {
	if key, ok := checkTransferWorldPossibleReasonArms[reason]; ok {
		return key
	}
	return CheckTransferWorldPossibleUnknownError
}

// checkTransferWorldPossibleReasonArms is the reason -> arm table as DATA, so
// a test can assert it covers the full required set rather than trusting a
// switch to have listed them.
var checkTransferWorldPossibleReasonArms = map[string]string{
	"world_same":        CheckTransferWorldPossibleUnknownError,
	"world_unknown":     CheckTransferWorldPossibleUnknownError,
	"world_full":        CheckTransferWorldPossibleUnknownError,
	"no_character_slot": CheckTransferWorldPossibleUnknownError,
	"banned":            CheckTransferWorldPossibleUnknownError,
	"is_guild_master":   CheckTransferWorldPossibleUnknownError,
	"is_gm":             CheckTransferWorldPossibleUnknownError,
	"in_family":         CheckTransferWorldPossibleInFamily,
	"trade_open":        CheckTransferWorldPossibleUnknownError,
	"merchant_open":     CheckTransferWorldPossibleUnknownError,
	"mts_listings_open": CheckTransferWorldPossibleUnknownError,
	"name_taken":        CheckTransferWorldPossibleUnknownError,
}

// CheckTransferWorldPossibleResultBody emits an arbitrary arm by its
// operations-table key — the general form the two builders above specialise.
// key must be one of the CheckTransferWorldPossible* constants; anything else
// resolves to ResolveCode's 99 sentinel, which the client renders as the
// unknown-error notice.
func CheckTransferWorldPossibleResultBody(characterId uint32, key string, birthDate uint32, worldNames []string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", key, func(code byte) packet.Encoder {
		return NewCheckTransferWorldPossibleResult(characterId, code, birthDate, worldNames)
	})
}
