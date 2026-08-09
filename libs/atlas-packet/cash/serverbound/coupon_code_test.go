package serverbound

import (
	"strings"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v48 ida=0x44d2e7
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v61 ida=0x45a6b5
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v72 ida=0x4698d8
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v79 ida=0x46aa3e
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v83 ida=0x4710e8
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v84 ida=0x473bde
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v87 ida=0x47b9d4
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v92 ida=0x484430
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v95 ida=0x487ee0
// packet-audit:verify packet=cash/serverbound/CouponCode version=jms_v185 ida=0x482450
func TestCouponCodeRoundTripSelfRedeem(t *testing.T) {
	// The plain-redeem path the client actually takes: targetCharacter empty, so
	// the conditional third string is never emitted on any version. jms_v185
	// still carries its unconditional nType byte between field 2 and the (here
	// absent) third string.
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CouponCode{targetCharacter: "", code: "MAPLE2026"}
			output := CouponCode{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacter() != "" {
				t.Errorf("targetCharacter: got %q, want empty", output.TargetCharacter())
			}
			if output.Code() != input.Code() {
				t.Errorf("code: got %q, want %q", output.Code(), input.Code())
			}
			if output.Extra() != "" {
				t.Errorf("extra: got %q, want empty", output.Extra())
			}
			if output.Type() != 0 {
				t.Errorf("type: got %d, want 0", output.Type())
			}
		})
	}
}

func TestCouponCodeRoundTripTargetedRedeem(t *testing.T) {
	// targetCharacter non-empty: gms_v48..v87 and jms_v185 add the guarded third
	// string; gms_v92/v95 have no third string at all, so extra decodes back
	// empty there. jms_v185 additionally round-trips nType.
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CouponCode{targetCharacter: "Sidekick", code: "MAPLE2026", nType: 7, extra: "EXTRA"}
			output := CouponCode{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacter() != input.TargetCharacter() {
				t.Errorf("targetCharacter: got %q, want %q", output.TargetCharacter(), input.TargetCharacter())
			}
			if output.Code() != input.Code() {
				t.Errorf("code: got %q, want %q", output.Code(), input.Code())
			}
			wantExtra := input.Extra()
			if !hasTrailingCouponString(ctx) {
				wantExtra = ""
			}
			if output.Extra() != wantExtra {
				t.Errorf("extra: got %q, want %q for %s", output.Extra(), wantExtra, v.Name)
			}
			wantType := input.Type()
			if !hasCouponType(ctx) {
				wantType = 0
			}
			if output.Type() != wantType {
				t.Errorf("type: got %d, want %d for %s", output.Type(), wantType, v.Name)
			}
		})
	}
}

// TestCouponCodeJmsCarriesTypeByte pins the jms_v185 divergence derived at
// COutPacket::Encode1 @ 0x4825e5: the nType byte sits between the coupon code
// and the guarded third string, and is on the wire unconditionally. Collapsing
// jms onto the GMS shape would desync the decoder by exactly one byte.
func TestCouponCodeJmsCarriesTypeByte(t *testing.T) {
	gms := pt.CreateContext("GMS", 83, 1)
	jms := pt.CreateContext("JMS", 185, 1)

	m := CouponCode{targetCharacter: "", code: "AB", nType: 9}
	gmsBytes := pt.Encode(t, gms, m.Encode, nil)
	jmsBytes := pt.Encode(t, jms, m.Encode, nil)

	if len(jmsBytes) != len(gmsBytes)+1 {
		t.Fatalf("jms body length %d, want gms length %d + 1", len(jmsBytes), len(gmsBytes))
	}
	if jmsBytes[len(jmsBytes)-1] != 9 {
		t.Errorf("jms trailing byte = %d, want the nType value 9", jmsBytes[len(jmsBytes)-1])
	}
}

// TestCouponCodeStringDoesNotLeakCode - coupon codes are secrets. String() must
// report the code's LENGTH and never its value.
func TestCouponCodeStringDoesNotLeakCode(t *testing.T) {
	m := CouponCode{targetCharacter: "Sidekick", code: "SUPERSECRET1"}
	s := m.String()
	if strings.Contains(s, "SUPERSECRET1") {
		t.Errorf("String() leaked the coupon code: %s", s)
	}
	if !strings.Contains(s, "12") {
		t.Errorf("String() = %q, want it to report the code length (12)", s)
	}
}

func TestCouponCodeOperation(t *testing.T) {
	if (CouponCode{}).Operation() != CashShopCouponCodeHandle {
		t.Errorf("Operation() = %q, want %q", (CouponCode{}).Operation(), CashShopCouponCodeHandle)
	}
	if CashShopCouponCodeHandle != "CashShopCouponCodeHandle" {
		t.Errorf("CashShopCouponCodeHandle = %q, want CashShopCouponCodeHandle", CashShopCouponCodeHandle)
	}
}
