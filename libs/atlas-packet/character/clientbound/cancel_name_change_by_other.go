package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CancelNameChangeByOtherWriter = "CancelNameChangeByOther"

// CancelNameChangeByOther is the clientbound CANCEL_NAME_CHANGE_BY_OTHER op —
// CWvsContext::OnCancelNameChangebyOther. It is the packet FR-2.7 requires
// when a pending name-change request was invalidated because another
// character claimed the target name first. It is a NOTIFICATION: no version
// has a serverbound counterpart that triggers it, and no cash-shop-state
// guard exists — the same OQ-9 finding as the two CANCEL_*_RESULT siblings.
//
// EMPTY payload — the client decodes NOTHING. Independently re-decompiled
// this pass on all seven applicable versions (derivation.md §2.8's addresses,
// re-verified directly rather than transcribed):
//
//	v72 0x922508, v79 0x97463d, v83 0xa2a7e6, v84 0xa75fa9 (func_query-confirmed;
//	pseudocode header still showed sub_A75FA9 — the documented rename-display
//	quirk), v87 0xac2482, v92 0x9cc170, v95 0x9f7620.
//
// Every one of the seven bodies is the identical shape: no `CInPacket::Decode*`
// call of any kind, just `StringPool::GetString` + `CUtilDlg::Notice` (a fixed
// StringPool id per version — v72/v79 393, v83 391, v84 394, v87 401, v92 406,
// v95 0x196 = 406) followed by clearing a `CWvsContext` cookie flag and
// stamping `get_update_time()`. There is no `nResult` byte, no `bHasMessage`
// flag, and NO sentinel bytes at all on this op — unlike the two
// CANCEL_*_RESULT siblings (whose three-way `0x00`/success-alt/otherwise
// shape switches this packet does NOT share), this row has no body to switch
// on. That is the finding this row's brief specifically warned might diverge
// from copy-sibling assumptions: it does, but in the direction of "no body",
// not a different sentinel pair.
//
// gms_v48, gms_v61 and jms_v185 are out of scope (confirmed this pass by
// direct `func_query` against each IDB, not assumed):
//   - v48: no `OnCancelNameChangebyOther` / `OnCancelNameChange*` match at
//     all — the name-change feature arrives with the family system at v61+
//     (matches the CANCEL_NAME_CHANGE_RESULT sibling's own v48 exclusion).
//   - v61: HAS `OnCancelNameChangeResult` (0x84ace9, the sibling op) but has
//     NO `OnCancelNameChangebyOther` match whatsoever — this "by other"
//     notification is not present at v61 even though its sibling cancel-result
//     packet is. This is the one exclusion this row does NOT share with its
//     siblings (they cover v61; this op does not).
//   - jms_v185: no match for `OnCancelNameChangebyOther`, `OnCancelNameChange*`,
//     or `CUICancelCharacterCoupon*` — jms_v185 has no name-change feature at
//     all (derivation.md §1.5), consistent with the whole family's jms
//     exclusion.
//
// Derivation: docs/tasks/task-227-cash-name-change-world-transfer/derivation.md §2.8.
// packet-audit:fname CWvsContext::OnCancelNameChangebyOther
type CancelNameChangeByOther struct{}

func NewCancelNameChangeByOther() CancelNameChangeByOther { return CancelNameChangeByOther{} }

func (m CancelNameChangeByOther) Operation() string { return CancelNameChangeByOtherWriter }
func (m CancelNameChangeByOther) String() string    { return "" }

// Encode writes nothing — the wire body is empty.
func (m CancelNameChangeByOther) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

// Decode reads nothing — the wire body is empty. Carried per the standing
// rule so the fixture test exercises the Decode path, not just Encode.
func (m *CancelNameChangeByOther) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
