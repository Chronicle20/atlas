package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CashShopCouponCodeHandle = "CashShopCouponCodeHandle"

// CouponCode - the standalone serverbound COUPON_CODE op. This is NOT a
// CASHSHOP_OPERATION mode arm: the client has no single cash-shop request
// builder, and the coupon submission gets its own opcode with NO leading mode
// byte (gms_v48 0xA1, v61 0xC5, v72 0xDC, v79 0xDE, v83 0xE6, v84 0xEC,
// v87 0xF3, v92 0x10D, v95 0x114, jms_v185 0xF6). The body begins immediately
// with strings.
//
// Field 1 is gms_v95's sCharacterID, the second out-parameter of
// CCouponUseSelectDlg::Confirm(&sCouponID, &sCharacterID) @ 0x487f5a — the
// character the coupon's reward is applied to. On the plain self-redeem path it
// is a zero-length string on every version (on gms_v48..v83 the dialog never
// writes it at all). Whether a populated value carries a character NAME or a
// numeric id as text is UNVERIFIED; nothing in Atlas consumes it today.
//
// Field 3 (`extra`) is emitted only when field 1 is non-empty (an
// `if (field1 && *field1)` guard in the client). It is present on gms_v48
// through gms_v87 and on jms_v185; gms_v92 and gms_v95 dropped it entirely
// (2 EncodeStr sites each). Its purpose is unknown / unverified.
//
// jms_v185 alone carries an extra 1-byte field between the coupon code and the
// guarded third string: an unconditional COutPacket::Encode1(nType) @ 0x4825e5,
// sourced from a control value the coupon dialog produces
// (CCashShop::ShowCouponInputDlg @ 0x482661). Its SEMANTICS are unknown /
// unverified — only its position, its width (1 byte) and its origin are
// established. It is modelled here so the decoder does not desync by one byte
// on every jms coupon submission.
//
// Derivation: docs/tasks/task-206-cash-shop-coupon-codes/derivation.md.
// packet-audit:fname CCashShop::OnStatusCoupon
type CouponCode struct {
	targetCharacter string
	code            string
	nType           byte
	extra           string
}

func (m CouponCode) TargetCharacter() string { return m.targetCharacter }
func (m CouponCode) Code() string            { return m.code }

// Type is the jms_v185-only byte described above. Its meaning is unverified; it
// is zero on every GMS version because those clients never put it on the wire.
func (m CouponCode) Type() byte    { return m.nType }
func (m CouponCode) Extra() string { return m.extra }

func (m CouponCode) Operation() string {
	return CashShopCouponCodeHandle
}

func (m CouponCode) String() string {
	// The coupon code is a secret; log its length, never its value.
	return fmt.Sprintf("targetCharacter [%s], code length [%d]", m.targetCharacter, len(m.code))
}

// hasTrailingCouponString reports whether this version's send path contains the
// third, conditionally-emitted EncodeStr. Derived per version: present on
// gms_v48/v61/v72/v79/v83/v84/v87 and on jms_v185; absent on gms_v92 and
// gms_v95, which have exactly 2 EncodeStr sites each.
func hasTrailingCouponString(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	if t.IsRegion("JMS") {
		return true
	}
	return t.IsRegion("GMS") && !t.MajorAtLeast(92)
}

// hasCouponType reports whether this version emits the unconditional 1-byte
// nType field between the coupon code and the guarded third string. Only
// jms_v185 does (Encode1 @ 0x4825e5); no GMS version has any Encode1 in this
// send path.
func hasCouponType(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.IsRegion("JMS")
}

func (m CouponCode) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.targetCharacter)
		w.WriteAsciiString(m.code)
		if hasCouponType(ctx) {
			w.WriteByte(m.nType)
		}
		if hasTrailingCouponString(ctx) && m.targetCharacter != "" {
			w.WriteAsciiString(m.extra)
		}
		return w.Bytes()
	}
}

func (m *CouponCode) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.targetCharacter = r.ReadAsciiString()
		m.code = r.ReadAsciiString()
		if hasCouponType(ctx) {
			m.nType = r.ReadByte()
		}
		if hasTrailingCouponString(ctx) && m.targetCharacter != "" {
			m.extra = r.ReadAsciiString()
		}
	}
}
