package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestAuthSuccessV48Body pins the gms_v48 LOGIN_STATUS (op 1) success wire.
//
// IDA-verified (GMS_v48_1_DEVM.exe, port 13337) — CLogin::OnCheckPasswordResult
// = sub_500931 @0x500931. Header: Decode1(result)@0x50095c, Decode1(regStatus)
// @0x500962, Decode4(GMS int)@0x500970. Success arm (result==0, regStatus<=1):
// Decode4(accountId)@0x500ef9, Decode1(gender)@0x500f01, Decode1(GM)@0x500f10,
// Decode1(admin)@0x500f18, DecodeStr(name)@0x500f21, Decode1(banReason)@0x500f36,
// Decode1(ban)@0x500f3e, DecodeBuffer(8)@0x500f49, DecodeBuffer(8)@0x500f67. The
// read STOPS at the second DecodeBuffer — there is NO trailing Decode4
// (nNumOfCharacter) that v61 @0x565f3e and up carry. v48 < 72 → no country byte;
// v48 > 12 → ban block; v48 < 83 → no pin/pic; v48 < 84 → no client key.
//
// packet-audit:verify packet=login/clientbound/AuthSuccess version=gms_v48 ida=0x500931
func TestAuthSuccessV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := AuthSuccess{accountId: 1001, name: "TestUser", gender: 1, usesPin: true, pic: "123456"}
	want := []byte{
		0x00,                   // Decode1 result @0x50095c
		0x00,                   // Decode1 regStatus @0x500962
		0x00, 0x00, 0x00, 0x00, // Decode4 GMS int @0x500970
		0xE9, 0x03, 0x00, 0x00, // Decode4 accountId=1001 @0x500ef9
		0x01, // Decode1 gender=1 @0x500f01
		0x00, // Decode1 GM(false) @0x500f10
		0x00, // Decode1 admin @0x500f18
		0x08, 0x00, 'T', 'e', 's', 't', 'U', 's', 'e', 'r', // DecodeStr name @0x500f21
		0x00,                                           // Decode1 banReason @0x500f36
		0x00,                                           // Decode1 ban @0x500f3e
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // DecodeBuffer(8) quiet ban ts @0x500f49
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // DecodeBuffer(8) creation ts @0x500f67
		// no nNumOfCharacter (absent < v61), no pin/pic (< 83), no client key (< 84)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 AuthSuccess body: got % x, want % x", got, want)
	}
}

// TestServerIPV48Body pins the gms_v48 SERVER_IP (op 12) success (migrate) wire.
//
// IDA-verified (GMS_v48_1_DEVM.exe, port 13337) — CLogin::OnSelectCharacterResult
// = sub_502B70 @0x502b70. Header: Decode1(code)@0x502b95, Decode1(mode)@0x502ba3.
// IP-connect arm (code==0 → LABEL_44): Decode4(ip)@0x502d61, Decode2(port)
// @0x502d6a, Decode4(clientId)@0x502d74, Decode1(bAuthenCode)@0x502d7f,
// Decode4(ulPremiumArgument)@0x502d86 (present, v48 > 12). Byte-for-byte the atlas
// ServerIP.Encode success block.
//
// packet-audit:verify packet=login/clientbound/ServerIP version=gms_v48 ida=0x502b70
func TestServerIPV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := ServerIP{code: 0, mode: 0, ipAddr: "192.168.1.1", port: 7575, clientId: 12345}
	want := []byte{
		0x00,                   // Decode1 code=0 @0x502b95
		0x00,                   // Decode1 mode=0 @0x502ba3
		0xC0, 0xA8, 0x01, 0x01, // Decode4 ip 192.168.1.1 @0x502d61
		0x97, 0x1D, // Decode2 port=7575 @0x502d6a
		0x39, 0x30, 0x00, 0x00, // Decode4 clientId=12345 @0x502d74
		0x00,                   // Decode1 bAuthenCode @0x502d7f
		0x00, 0x00, 0x00, 0x00, // Decode4 ulPremiumArgument @0x502d86 (v48 > 12)
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 ServerIP body: got % x, want % x", got, want)
	}
}
