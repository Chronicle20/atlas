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

const CashShopCancelNameChangeResultWriter = "CashShopCancelNameChangeResult"

// Reason-code keys for the tenant socket-config `operations` table of the
// CashShopCancelNameChangeResult writer. The VALUES live in the eight
// applicable seed templates, never here (DOM-25).
//
// Unlike the CheckNameChangePossibleResult / CheckTransferWorldPossibleResult
// enumerations, this is NOT a flat switch on the whole result byte — it is a
// three-way SHAPE switch (derivation.md §2.6, independently re-decompiled this
// pass on gms_v83 @0xa2a677 and gms_v61 @0x84ace9, both byte-for-byte
// identical):
//
//	nResult == 0x00   CUICancelCharacterCouponResults(0), modal, no further read
//	nResult == 0xFF   CUICancelCharacterCouponResults(1), modal, no further read
//	otherwise         Decode1 bHasMessage
//	                     if bHasMessage: DecodeStr sMessage -> Notice(sMessage)
//	                     else:                                 Notice(StringPool unknown-error)
//
// Only the two success sentinels are given operations-table keys: 0x00 and
// 0xFF are exactly the values every one of the eight receivers compares
// against (v61 0x84ace9 … v95 0xa01b10, all structurally identical). The
// failure arm is not a single wire value — it is "anything that is not 0x00 or
// 0xFF" — so CancelNameChangeResultFailed resolves to a representative byte
// (the templates configure 1, matching the CANCELLED_ALT/0xFF asymmetry note
// below) purely so the byte is config-resolved rather than a Go literal; any
// value outside {0x00, 0xFF} produces the identical failure shape on every
// applicable version.
//
// StringPool id for the unknown-error Notice, per version (derivation.md
// §2.6): v61/v72 3492/3546 (sic — v61 uses 3492, v72 uses 3546) · v79 3550 ·
// v83 SP_3570 · v84 3573 · v87 3579 · v92 3639 · v95 0xE16 (3606). Not
// reproduced here as a value: the string is rendered client-side from the id
// baked into the client binary, never sent on the wire — only bHasMessage and
// the optional literal sMessage travel on this packet's failure arm.
//
// The 0xFF / 0x01 asymmetry against the sibling CANCEL_TRANSFER_WORLD_RESULT
// (whose second success sentinel is 0x01, not 0xFF) is the single easiest way
// to write two decoders that both pass a fixture and are silently wrong
// (derivation.md §2.7); this codec is deliberately NOT shared with that op.
const (
	// CancelNameChangeResultCancelled is the nResult == 0x00 arm.
	CancelNameChangeResultCancelled = "CANCELLED"
	// CancelNameChangeResultCancelledAlt is the nResult == 0xFF arm — a second,
	// client-distinguished "cancelled" variant (CUICancelCharacterCouponResults(1)
	// vs (0)). Both are "cancelled" on the wire; the game-logic caller decides
	// which one to emit.
	CancelNameChangeResultCancelledAlt = "CANCELLED_ALT"
	// CancelNameChangeResultFailed is any nResult outside {0x00, 0xFF} — the
	// message-carrying failure shape. See the type doc comment for why this key
	// resolves to a representative byte rather than one fixed wire value.
	CancelNameChangeResultFailed = "FAILED"
)

// CancelNameChangeResult is the clientbound CANCEL_NAME_CHANGE_RESULT op —
// CWvsContext::OnCancelNameChangeResult. It is a NOTIFICATION: no version has
// a serverbound counterpart that triggers it (derivation.md §2.6; nothing in
// any of the eight IDBs sends a request this packet answers). Decode is
// carried anyway, per the standing rule, so the fixture test can prove
// byte-exactness independent of Encode.
//
// Body, IDENTICAL on every applicable version — v61, v72, v79, v83, v84, v87,
// v92, v95 (derivation.md §2.6, re-decompiled this pass on v61 and v83):
//
//	Decode1  nResult
//	  0x00 -> CUICancelCharacterCouponResults(0), modal, no further read
//	  0xFF -> CUICancelCharacterCouponResults(1), modal, no further read
//	  otherwise:
//	    Decode1  bHasMessage
//	      if bHasMessage: DecodeStr sMessage
//	      (else nothing further)
//
// Receiver addresses (derivation.md §2.6): v61 0x84ace9, v72 0x922399,
// v79 0x9744ce, v83 0xa2a677, v84 0xa75e3a, v87 0xac2313, v92 0x9d64a0,
// v95 0xa01b10.
//
// Absent from gms_v48 and jms_v185: v48 has no OnCancelNameChangeResult
// receiver at all (the name-change feature and its cancel flow both arrive
// with the family system at v61+), and jms_v185 has no name-change feature
// whatsoever (derivation.md §1.5) — its two CANCEL_* receivers cover only the
// world-transfer flow.
//
// OQ-9 (does the client accept this packet outside the cash-shop UI?):
// answered NO evidence of any state guard. CWvsContext::OnCancelNameChangeResult
// (v83 @0xa2a677, v61 @0x84ace9) opens its modal / renders its Notice
// unconditionally — there is no read of a cash-shop-open flag, no early
// return, and no CCashShop member access anywhere in the decompiled body on
// either version checked. CWvsContext is the always-live session-context
// singleton (the same class CANCEL_NAME_CHANGE_BY_OTHER and every other
// account-notification packet targets), not a UI-scoped dialog controller, so
// the packet is processed whenever CWvsContext::OnPacket dispatches it,
// cash-shop UI open or not. This does not change any behaviour in this task
// (design §3.9's pink-text belt-and-braces sends regardless), but the finding
// is recorded here per the brief rather than left open.
//
// Derivation: docs/tasks/task-227-cash-name-change-world-transfer/derivation.md.
// packet-audit:fname CWvsContext::OnCancelNameChangeResult
type CancelNameChangeResult struct {
	result     byte
	hasMessage bool
	message    string
}

func NewCancelNameChangeResult(result byte, message string) CancelNameChangeResult {
	return CancelNameChangeResult{result: result, hasMessage: message != "", message: message}
}

// Result is the resolved wire byte, NOT a semantic code. It comes from the
// tenant template's operations table via one of the body builders below.
func (m CancelNameChangeResult) Result() byte { return m.result }

// HasMessage is the bHasMessage wire flag. Only meaningful (and only present
// on the wire) when Result is outside {0x00, 0xFF}.
func (m CancelNameChangeResult) HasMessage() bool { return m.hasMessage }

// Message is the optional sMessage the client Notice()s verbatim on the
// failure arm. Empty when HasMessage is false.
func (m CancelNameChangeResult) Message() string { return m.message }

func (m CancelNameChangeResult) Operation() string {
	return CashShopCancelNameChangeResultWriter
}

func (m CancelNameChangeResult) String() string {
	return fmt.Sprintf("result [%d], hasMessage [%t], message [%s]", m.result, m.hasMessage, m.message)
}

// cancelNameChangeResultIsShapeSwitch reports whether the wire result byte is
// one of the two success sentinels (which read no further bytes) or the
// message-carrying failure shape (which does). Byte-identical across every
// applicable version — derivation.md §2.6 shows no version delta.
func cancelNameChangeResultIsShapeSwitch(result byte) bool {
	return result != 0x00 && result != 0xFF
}

func (m CancelNameChangeResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.result)
		if cancelNameChangeResultIsShapeSwitch(m.result) {
			w.WriteBool(m.hasMessage)
			if m.hasMessage {
				w.WriteAsciiString(m.message)
			}
		}
		return w.Bytes()
	}
}

func (m *CancelNameChangeResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.result = r.ReadByte()
		if cancelNameChangeResultIsShapeSwitch(m.result) {
			m.hasMessage = r.ReadBool()
			if m.hasMessage {
				m.message = r.ReadAsciiString()
			}
		}
	}
}

// CancelNameChangeResultCancelledBody emits the nResult == 0x00 arm.
func CancelNameChangeResultCancelledBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CancelNameChangeResultCancelled, func(code byte) packet.Encoder {
		return NewCancelNameChangeResult(code, "")
	})
}

// CancelNameChangeResultCancelledAltBody emits the nResult == 0xFF arm.
func CancelNameChangeResultCancelledAltBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CancelNameChangeResultCancelledAlt, func(code byte) packet.Encoder {
		return NewCancelNameChangeResult(code, "")
	})
}

// CancelNameChangeResultFailedBody emits the message-carrying failure arm.
// message may be empty, in which case bHasMessage is false and the client
// renders its own built-in unknown-error string instead.
func CancelNameChangeResultFailedBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CancelNameChangeResultFailed, func(code byte) packet.Encoder {
		return NewCancelNameChangeResult(code, message)
	})
}
