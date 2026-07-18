package clientbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/clientbound/AuthSuccess version=gms_v83 ida=0x5f83ee
// packet-audit:verify packet=login/clientbound/AuthSuccess version=gms_v87 ida=0x62fb84
// packet-audit:verify packet=login/clientbound/AuthSuccess version=gms_v95 ida=0x5dc600
// packet-audit:verify packet=login/clientbound/AuthSuccess version=gms_v84 ida=0x60d368
// packet-audit:verify packet=login/clientbound/AuthSuccess version=jms_v185 ida=0x66e79f
// packet-audit:verify packet=login/clientbound/AuthSuccess version=gms_v79 ida=0x5cd38f
//
// gms_v72 LOGIN_STATUS success path — CLogin::OnCheckPasswordResult @0x5b2577,
// success branch @0x5b2c40 (GMS_v72.1_U_DEVM.exe, port 13339): Decode4(accountId)
// @0x5b2c4c, Decode1(gender)@0x5b2c54, Decode1(gm)@0x5b2c63, Decode1(admin)
// @0x5b2c6d, Decode1(country)@0x5b2c75, DecodeStr(name)@0x5b2c7e, Decode1
// @0x5b2c93, Decode1@0x5b2c9b, DecodeBuffer(8)@0x5b2ca6, DecodeBuffer(8)
// @0x5b2cc4, Decode4(nNumOfCharacter)@0x5b2ce1 — NO pin/pic flags (<83), NO
// client key (<84). Byte-for-byte the atlas AuthSuccess.Encode GMS legacy path.
//
// gms_v61 LOGIN_STATUS success path — CLogin::OnCheckPasswordResult @0x5657ce,
// success branch @0x565ea4 (GMS_v61.1_U_DEVM.exe, port 13338): Decode4(accountId)
// @0x565eb0, Decode1(gender)@0x565eb8, Decode1(GM)@0x565ec7, Decode1(admin)
// @0x565ecf, DecodeStr(name)@0x565ed8, Decode1@0x565eed, Decode1@0x565ef5,
// DecodeBuffer(8)@0x565f00, DecodeBuffer(8)@0x565f1e, Decode4(nNumOfCharacter)
// @0x565f3e. Only 3 bytes between accountId and name (gender/GM/admin) — the v72
// country byte is ABSENT (v61 < 72). No pin/pic (<83), no client key (<84).
// Width = 46 (v72) − 1 (country) = 45 bytes.
//
// packet-audit:verify packet=login/clientbound/AuthSuccess version=gms_v61 ida=0x5657ce
func TestAuthSuccessV61WireWidth(t *testing.T) {
	// result(1)+byte(1)+GMSint(4)+accountId(4)+gender(1)+gmBool(1)+admin(1)+
	// (2+len name)+banReason(1)+ban(1)+long(8)+long(8)+nNumOfChar(4)
	// = 1+1+4+4+1+1+1+(2+8)+1+1+8+8+4 = 45 bytes (no country, no pin/pic, no key).
	const wantLen = 45
	ctx := pt.CreateContext("GMS", 61, 1)
	input := AuthSuccess{accountId: 1001, name: "TestUser", gender: 1, usesPin: true, pic: "123456"}
	l, _ := testlog.NewNullLogger()
	if got := len(input.Encode(l, ctx)(nil)); got != wantLen {
		t.Fatalf("v61 wire len: got %d, want %d", got, wantLen)
	}
}

// packet-audit:verify packet=login/clientbound/AuthSuccess version=gms_v72 ida=0x5b2577
func TestAuthSuccessV72WireWidth(t *testing.T) {
	// result(1)+byte(1)+GMSint(4)+accountId(4)+gender(1)+gmBool(1)+admin(1)+
	// country(1)+(2+len name)+banReason(1)+ban(1)+long(8)+long(8)+nNumOfChar(4)
	// = 1+1+4+4+1+1+1+1+(2+8)+1+1+8+8+4 = 46 bytes (no pin/pic, no client key).
	const wantLen = 46
	ctx := pt.CreateContext("GMS", 72, 1)
	input := AuthSuccess{accountId: 1001, name: "TestUser", gender: 1, usesPin: true, pic: "123456"}
	l, _ := testlog.NewNullLogger()
	if got := len(input.Encode(l, ctx)(nil)); got != wantLen {
		t.Fatalf("v72 wire len: got %d, want %d", got, wantLen)
	}
}

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
			// pin/pic flags exist only at GMS v83+ (IDA v79 OnCheckPasswordResult
			// @0x5cd38f reads none; introduced at v83).
			if v.Region == "GMS" && v.MajorVersion >= 83 {
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
