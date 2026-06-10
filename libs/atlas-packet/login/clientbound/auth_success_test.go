package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestAuthSuccessV95WireWidthMatchesIDA(t *testing.T) {
	// Spike: docs/packets/spike-login-v95.md Packet 1.
	// Field 7 (subGradeCode+testerAccount) is int16 in v95, byte before.
	// Per-row width sum for input {accountId:1001, name:"TestUser", gender:1, usesPin:true, pic:"123456"}:
	//   byte+byte+int32+int32+byte+byte+int16+byte+(2+len("TestUser"))+byte+byte+int64+int64+int32+byte+byte+int64
	// = 1+1+4+4+1+1+2+1+(2+8)+1+1+8+8+4+1+1+8 = 57 bytes
	const wantLen = 57

	ctx := pt.CreateContext("GMS", 95, 1)
	input := AuthSuccess{
		accountId: 1001,
		name:      "TestUser",
		gender:    1,
		usesPin:   true,
		pic:       "123456",
	}
	l, _ := testlog.NewNullLogger()
	bytes := input.Encode(l, ctx)(nil)
	if len(bytes) != wantLen {
		t.Fatalf("v95 wire len: got %d, want %d", len(bytes), wantLen)
	}
}

func TestAuthSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AuthSuccess{
				accountId: 1001,
				name:      "TestUser",
				gender:    1,
				usesPin:   true,
				pic:       "123456",
			}
			output := AuthSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.AccountId() != input.AccountId() {
				t.Errorf("accountId: got %v, want %v", output.AccountId(), input.AccountId())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Gender() != input.Gender() {
				t.Errorf("gender: got %v, want %v", output.Gender(), input.Gender())
			}
			if v.Region == "GMS" && v.MajorVersion > 12 {
				if output.UsesPin() != input.UsesPin() {
					t.Errorf("usesPin: got %v, want %v", output.UsesPin(), input.UsesPin())
				}
			}
		})
	}
}

// TestAuthSuccessClientKeyBoundary pins the 8-byte client key
// (CWvsContext::m_aClientKey[8]). The GMS client reads it unconditionally on
// OnCheckPasswordResult's success path, gated >83 in the client binary —
// present in v84/v87/v95, absent in v83. atlas previously gated the write
// >=87, so a v84/85/86 client underran the packet -> CInPacket throws
// ZException(38) -> silent disconnect before the world list renders. The key
// must therefore be present for GMS v84+. v83 stays keyless; v87/v95 unchanged.
// (Found via the live v84 client; the server self-round-trip could not catch it
// because Decode mirrored the same wrong gate.)
func TestAuthSuccessClientKeyBoundary(t *testing.T) {
	enc := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		in := AuthSuccess{accountId: 1001, name: "TestUser", gender: 1, usesPin: true, pic: "123456"}
		l, _ := testlog.NewNullLogger()
		return in.Encode(l, ctx)(nil)
	}
	v83 := enc(83)
	// v84..86 (the previously-broken gap) and v87 all carry the 8-byte key, so
	// each is exactly 8 bytes longer than the keyless v83 encoding.
	for _, major := range []uint16{84, 85, 86, 87} {
		if got := enc(major); len(got) != len(v83)+8 {
			t.Errorf("AuthSuccess GMS v%d encoded len %d; want v83 len %d + 8 (m_aClientKey[8])", major, len(got), len(v83))
		}
	}
}
