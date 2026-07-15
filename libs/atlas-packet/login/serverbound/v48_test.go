package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestAfterLoginV48LegacyWire pins the gms_v48 AFTER_LOGIN (COutPacket op 8)
// serverbound wire: the LEGACY shape carrying the accountId int between opt2 and
// the pin string.
//
// IDA-verified (GMS_v48_1_DEVM.exe, port 13337) — CLogin::OnCheckPinCodeResult
// = sub_503956 @0x503956. The Decode1(dialogResult)==2 arm @0x503ae3 builds
// COutPacket(8): Encode1(pinResult)@0x503af0, then when set Encode1(0)@0x503b0c,
// Encode4(accountId = off_80C8A0[2057])@0x503b1f, EncodeStr(pin)@0x503b62. (The
// ==4 arm @0x5039f3 builds the same op-8 body.) atlas gates the accountId int to
// GMS<83, so v48 (<83) emits it — byte-for-byte the v61 legacy wire.
//
// packet-audit:verify packet=login/serverbound/AfterLogin version=gms_v48 ida=0x503956
func TestAfterLoginV48LegacyWire(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	// accountId 0x04030201 -> LE 01 02 03 04; pin "ab" -> WriteShort(2 LE) 02 00 + 'a''b'.
	input := AfterLogin{pinMode: 5, opt2: 7, accountId: 0x04030201, pin: "ab"}
	want := []byte{
		0x05,                   // Encode1 pinMode=5 @0x503af0
		0x07,                   // Encode1 opt2=0 (here 7) @0x503b0c
		0x01, 0x02, 0x03, 0x04, // Encode4 accountId @0x503b1f (GMS<83 legacy)
		0x02, 0x00, 'a', 'b', // EncodeStr pin @0x503b62
	}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v48 AfterLogin body: got % x, want % x", got, want)
	}
}

// TestAllCharacterListRequestV48 pins the gms_v48 VIEW_ALL_CHAR (op 12)
// serverbound wire: an EMPTY body.
//
// IDA-verified (GMS_v48_1_DEVM.exe, port 13337) — the view-all-char send
// sub_502293 @0x502293 builds a bare COutPacket(12) @0x5022e1 then SendPacket
// @0x5022f3 with NO Encode* calls between → zero-length body (it first sends the
// bare migrate op 11 @0x5022b6 when this[72]!=1, then the empty op 12). atlas
// AllCharacterListRequest.Encode only emits the extra block for GMS>=87, so for
// v48 (<87) it writes nothing.
//
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v48 ida=0x502293
func TestAllCharacterListRequestV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := AllCharacterListRequest{gameStartMode: 1, nexonPassport: "passport", machineId: make([]byte, 16), gameRoomClient: 42, gameStartMode2: 2}
	if got := pt.Encode(t, ctx, input.Encode, nil); len(got) != 0 {
		t.Errorf("v48 AllCharacterListRequest body: got % x, want empty", got)
	}
}
