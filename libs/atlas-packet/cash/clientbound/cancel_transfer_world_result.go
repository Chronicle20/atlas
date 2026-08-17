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

const CashShopCancelTransferWorldResultWriter = "CashShopCancelTransferWorldResult"

// Reason-code keys for the tenant socket-config `operations` table of the
// CashShopCancelTransferWorldResult writer. The VALUES live in the eight
// applicable seed templates, never here (DOM-25).
//
// Same three-way SHAPE switch as the sibling CancelNameChangeResult
// (derivation.md §2.7, independently re-decompiled this pass on gms_v61
// @0x84ae56, gms_v83 @0xa2a82d, gms_v92 @0x9d6680 and gms_v95 @0xa01cf0 — all
// four byte-for-byte identical in shape):
//
//	nResult == 0x00   CUICancelCharacterCouponResults(2), modal, no further read
//	nResult == 0x01   CUICancelCharacterCouponResults(3), modal, no further read
//	otherwise         Decode1 bHasMessage
//	                     if bHasMessage: DecodeStr sMessage -> Notice(sMessage)
//	                     else:                                 Notice(StringPool unknown-error)
//
// The 0xFF / 0x01 asymmetry against the sibling CANCEL_NAME_CHANGE_RESULT
// (whose second success sentinel is 0xFF, not 0x01) is the single easiest
// way to write two decoders that both pass a fixture and are silently wrong
// (derivation.md §2.7, holds on all eight versions); this codec is
// deliberately NOT shared with that op — and the FAILED arm's representative
// operations-table byte below is chosen to sit clear of BOTH sentinels
// (0x00, 0x01, 0xFF) rather than reusing the sibling's happen-to-work value.
//
// Only the two success sentinels are given operations-table keys: 0x00 and
// 0x01 are exactly the values every one of the eight receivers compares
// against (v61 0x84ae56 … v95 0xa01cf0, all structurally identical). The
// failure arm is not a single wire value — it is "anything that is not 0x00
// or 0x01" — so CancelTransferWorldResultFailed resolves to a representative
// byte purely so the byte is config-resolved rather than a Go literal; any
// value outside {0x00, 0x01} produces the identical failure shape on every
// applicable version.
const (
	// CancelTransferWorldResultCancelled is the nResult == 0x00 arm.
	CancelTransferWorldResultCancelled = "CANCELLED"
	// CancelTransferWorldResultCancelledAlt is the nResult == 0x01 arm — a
	// second, client-distinguished "cancelled" variant
	// (CUICancelCharacterCouponResults(3) vs (2)). Both are "cancelled" on
	// the wire; the game-logic caller decides which one to emit.
	CancelTransferWorldResultCancelledAlt = "CANCELLED_ALT"
	// CancelTransferWorldResultFailed is any nResult outside {0x00, 0x01} —
	// the message-carrying failure shape. See the type doc comment for why
	// this key resolves to a representative byte rather than one fixed wire
	// value.
	CancelTransferWorldResultFailed = "FAILED"
)

// CancelTransferWorldResult is the clientbound CANCEL_TRANSFER_WORLD_RESULT
// op — CWvsContext::OnCancelTransferWorldResult. It is a NOTIFICATION: no
// version has a serverbound counterpart that triggers it (derivation.md
// §2.7; nothing in any of the eight IDBs sends a request this packet
// answers). Decode is carried anyway, per the standing rule, so the fixture
// test can prove byte-exactness independent of Encode.
//
// Body, IDENTICAL on every applicable version — v61, v72, v79, v83, v84,
// v87, v92, v95 (derivation.md §2.7, re-decompiled this pass on v61, v83,
// v92 and v95):
//
//	Decode1  nResult
//	  0x00 -> CUICancelCharacterCouponResults(2), modal, no further read
//	  0x01 -> CUICancelCharacterCouponResults(3), modal, no further read
//	  otherwise:
//	    Decode1  bHasMessage
//	      if bHasMessage: DecodeStr sMessage
//	      (else nothing further)
//
// Receiver addresses (derivation.md §2.7): v61 0x84ae56, v72 0x92254f,
// v79 0x974684, v83 0xa2a82d, v84 0xa75ff0, v87 0xac24c9, v92 0x9d6680,
// v95 0xa01cf0.
//
// Absent from gms_v48 and jms_v185, both confirmed by direct func_query
// against the respective IDBs this pass (not assumed): neither has an
// "*OnCancelTransferWorldResult*" match, nor any "*CUICancelCharacterCoupon*"
// / "*OnCancel*" match that could be a differently-named equivalent. v48
// does carry the world-transfer buy/check flow
// (CCashShop::OnCheckTransferWorldPossibleResult @0x455d25,
// OnCashItemResTransferWorldDone/Failed) but no cancel-result receiver at
// all — the cancel flow arrives with the family system at v61+, same as the
// sibling name-change flow. jms_v185 likewise carries the world-transfer buy
// flow (CCashShop::OnCheckTransferWorldPossibleResult @0x48e7a6,
// CUITransferWorldLicenseNotice/CUITransferWorldSelectDlg) but has no
// CWvsContext-level cancel-result receiver and no CUICancelCharacterCoupon*
// dialog type anywhere in the binary — the "cancel a pending coupon
// purchase" UX this op serves simply doesn't exist on jms_v185, matching
// derivation.md §1.5's finding that jms_v185 lacks the whole
// CUICancelCharacterCouponResults family (it covers both CANCEL_* rows, not
// just name-change).
//
// OQ-9 (does the client accept this packet outside the cash-shop UI?):
// same finding as the sibling CANCEL_NAME_CHANGE_RESULT — no cash-shop-state
// guard. CWvsContext::OnCancelTransferWorldResult (v83 @0xa2a82d, v61
// @0x84ae56) opens its modal / renders its Notice unconditionally: no read
// of a cash-shop-open flag, no early return, no CCashShop member access on
// either version checked.
//
// Derivation: docs/tasks/task-227-cash-name-change-world-transfer/derivation.md.
// packet-audit:fname CWvsContext::OnCancelTransferWorldResult
type CancelTransferWorldResult struct {
	result     byte
	hasMessage bool
	message    string
}

func NewCancelTransferWorldResult(result byte, message string) CancelTransferWorldResult {
	return CancelTransferWorldResult{result: result, hasMessage: message != "", message: message}
}

// Result is the resolved wire byte, NOT a semantic code. It comes from the
// tenant template's operations table via one of the body builders below.
func (m CancelTransferWorldResult) Result() byte { return m.result }

// HasMessage is the bHasMessage wire flag. Only meaningful (and only present
// on the wire) when Result is outside {0x00, 0x01}.
func (m CancelTransferWorldResult) HasMessage() bool { return m.hasMessage }

// Message is the optional sMessage the client Notice()s verbatim on the
// failure arm. Empty when HasMessage is false.
func (m CancelTransferWorldResult) Message() string { return m.message }

func (m CancelTransferWorldResult) Operation() string {
	return CashShopCancelTransferWorldResultWriter
}

func (m CancelTransferWorldResult) String() string {
	return fmt.Sprintf("result [%d], hasMessage [%t], message [%s]", m.result, m.hasMessage, m.message)
}

// cancelTransferWorldResultIsShapeSwitch reports whether the wire result
// byte is one of the two success sentinels (which read no further bytes) or
// the message-carrying failure shape (which does). Byte-identical across
// every applicable version — derivation.md §2.7 shows no version delta.
func cancelTransferWorldResultIsShapeSwitch(result byte) bool {
	return result != 0x00 && result != 0x01
}

func (m CancelTransferWorldResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.result)
		if cancelTransferWorldResultIsShapeSwitch(m.result) {
			w.WriteBool(m.hasMessage)
			if m.hasMessage {
				w.WriteAsciiString(m.message)
			}
		}
		return w.Bytes()
	}
}

func (m *CancelTransferWorldResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadByte()
		if cancelTransferWorldResultIsShapeSwitch(m.result) {
			m.hasMessage = r.ReadBool()
			if m.hasMessage {
				m.message = r.ReadAsciiString()
			}
		}
	}
}

// CancelTransferWorldResultCancelledBody emits the nResult == 0x00 arm.
func CancelTransferWorldResultCancelledBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CancelTransferWorldResultCancelled, func(code byte) packet.Encoder {
		return NewCancelTransferWorldResult(code, "")
	})
}

// CancelTransferWorldResultCancelledAltBody emits the nResult == 0x01 arm.
func CancelTransferWorldResultCancelledAltBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CancelTransferWorldResultCancelledAlt, func(code byte) packet.Encoder {
		return NewCancelTransferWorldResult(code, "")
	})
}

// CancelTransferWorldResultFailedBody emits the message-carrying failure
// arm. message may be empty, in which case bHasMessage is false and the
// client renders its own built-in unknown-error string instead.
func CancelTransferWorldResultFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CancelTransferWorldResultFailed, func(code byte) packet.Encoder {
		return NewCancelTransferWorldResult(code, message)
	})
}
