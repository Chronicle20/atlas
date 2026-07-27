package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestAllCharacterListRequestV79 pins the gms_v79 VIEW_ALL_CHAR (op 13)
// serverbound wire: an EMPTY body.
//
// IDA-verified (GMS_v79_1_DEVM.exe, port 13340) — the view-all-char send
// sub_5CEDE1 @0x5CEDE1 builds COutPacket(13) @0x5cee2b and SendPacket
// @0x5cee3d with NO Encode* calls between → zero-length body. atlas
// AllCharacterListRequest.Encode only emits the extra block for GMS>=87, so for
// v79 (<87) it writes nothing — matching the empty client send.
//
// The gms_v79 export entry for CLogin::SendViewAllCharPacket was an unresolved
// stub (the live v79 IDB names the sender sub_5CEDE1); it was surgically spliced
// with addr 0x5cede1 + empty-body calls (COutPacket(13) with no Encode) so the
// citation resolves. Report is FlatInvalid (the version-gated >=87 fields read
// as "extra" for v79) — the same tolerated shape as the verified v83 cell.
//
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v79 ida=0x5cede1
func TestAllCharacterListRequestV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := AllCharacterListRequest{gameStartMode: 1, nexonPassport: "passport", machineId: make([]byte, 16), gameRoomClient: 42, gameStartMode2: 2}
	if got := pt.Encode(t, ctx, input.Encode, nil); len(got) != 0 {
		t.Errorf("v79 AllCharacterListRequest body: got % x, want empty", got)
	}
}

// TestAllCharacterListRequestV72 pins the gms_v72 VIEW_ALL_CHAR (op 13)
// serverbound wire: an EMPTY body.
//
// IDA-verified (GMS_v72.1_U_DEVM.exe, port 13339) — CLogin::SendViewAllCharPacket
// = sub_5B3EE7 @0x5b3ee7 builds COutPacket(13) @0x5b3f31 then SendPacket @0x5b3f43
// with NO Encode* calls between → zero-length body. atlas
// AllCharacterListRequest.Encode only emits the extra block for GMS>=87, so for
// v72 (<87) it writes nothing — matching the empty client send.
//
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v72 ida=0x5b3ee7
func TestAllCharacterListRequestV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := AllCharacterListRequest{gameStartMode: 1, nexonPassport: "passport", machineId: make([]byte, 16), gameRoomClient: 42, gameStartMode2: 2}
	if got := pt.Encode(t, ctx, input.Encode, nil); len(got) != 0 {
		t.Errorf("v72 AllCharacterListRequest body: got % x, want empty", got)
	}
}

// TestAllCharacterListRequestV61 pins the gms_v61 VIEW_ALL_CHAR (op 13)
// serverbound wire: an EMPTY body.
//
// IDA-verified (GMS_v61.1_U_DEVM.exe, port 13338) — the view-all-char send
// sub_567117 @0x567117 builds COutPacket(13)@0x567165 then SendPacket@0x567177
// with NO Encode* calls between → zero-length body (it first sends op 12 when
// this[79]!=1, then the empty op 13). atlas AllCharacterListRequest.Encode only
// emits the extra block for GMS>=87, so for v61 (<87) it writes nothing.
//
// The gms_v61 export entry for CLogin::SendViewAllCharPacket was an unresolved
// stub (the live v61 IDB names the sender sub_567117); it was surgically spliced
// with addr 0x567117 + empty-body calls so the citation resolves (same shape as
// the verified v79 cell).
//
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v61 ida=0x567117
func TestAllCharacterListRequestV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := AllCharacterListRequest{gameStartMode: 1, nexonPassport: "passport", machineId: make([]byte, 16), gameRoomClient: 42, gameStartMode2: 2}
	if got := pt.Encode(t, ctx, input.Encode, nil); len(got) != 0 {
		t.Errorf("v61 AllCharacterListRequest body: got % x, want empty", got)
	}
}

// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v83 ida=0x5fac34
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v87 ida=0x6324e3
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v95 ida=0x5dfb40
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v84 ida=0x60fc30
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=jms_v185 ida=0x67094e
func TestAllCharacterListRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AllCharacterListRequest{
				gameStartMode:  1,
				nexonPassport:  "passport",
				machineId:      make([]byte, 16),
				gameRoomClient: 42,
				gameStartMode2: 2,
			}
			output := AllCharacterListRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			// Only encoded for GMS >= 87 (v84..86 == v83, off-by-one fix, delta §3.2)
			if v.Region == "GMS" && v.MajorVersion >= 87 {
				if output.GameStartMode() != input.GameStartMode() {
					t.Errorf("gameStartMode: got %v, want %v", output.GameStartMode(), input.GameStartMode())
				}
				if output.NexonPassport() != input.NexonPassport() {
					t.Errorf("nexonPassport: got %v, want %v", output.NexonPassport(), input.NexonPassport())
				}
				if output.GameRoomClient() != input.GameRoomClient() {
					t.Errorf("gameRoomClient: got %v, want %v", output.GameRoomClient(), input.GameRoomClient())
				}
				if output.GameStartMode2() != input.GameStartMode2() {
					t.Errorf("gameStartMode2: got %v, want %v", output.GameStartMode2(), input.GameStartMode2())
				}
			}
		})
	}
}
