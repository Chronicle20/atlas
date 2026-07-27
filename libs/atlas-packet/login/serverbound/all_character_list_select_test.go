package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61: PICK_ALL_CHAR send = sub_5650B6 @0x5650b6 (GMS_v61.1_U_DEVM.exe, port
// 13338): COutPacket(14)@0x5651e0; Encode4(charId)@0x565201; Encode4(worldId)
// @0x565218; EncodeStr(mac=GetLocalMacAddress)@0x565253; EncodeStr(hwid=
// GetLocalMacAddressWithHDDSerialNo)@0x565289. Matches the codec exactly.
//
// packet-audit:verify packet=login/serverbound/AllCharacterListSelect version=gms_v61 ida=0x5650b6
func TestAllCharacterListSelectV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := AllCharacterListSelect{characterId: 12345, worldId: world.Id(2), mac: "AA:BB:CC", hwid: "HWID"}
	want := append(le4(12345), le4(2)...)
	want = append(want, lp("AA:BB:CC")...)
	want = append(want, lp("HWID")...)
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 AllCharacterListSelect body: got % x, want % x", got, want)
	}
}

// gms_v72: CLogin::SendSelectCharPacketByVAC = sub_5B1E3F @0x5b1e3f
// (GMS_v72.1_U_DEVM.exe, port 13339): COutPacket(14) @0x5b1f68; Encode4(charId)
// @0x5b1f89; Encode4(worldId) @0x5b1fa0; EncodeStr(mac) @0x5b1fdb; EncodeStr(hwid)
// @0x5b2011. Matches the codec. Marker-only (tier-0).
//
// packet-audit:verify packet=login/serverbound/AllCharacterListSelect version=gms_v72 ida=0x5b1e3f
// packet-audit:verify packet=login/serverbound/AllCharacterListSelect version=gms_v83 ida=0x5f76ae
// packet-audit:verify packet=login/serverbound/AllCharacterListSelect version=gms_v84 ida=0x60c624
// packet-audit:verify packet=login/serverbound/AllCharacterListSelect version=gms_v87 ida=0x62ee37
// packet-audit:verify packet=login/serverbound/AllCharacterListSelect version=gms_v95 ida=0x5d7550
// packet-audit:verify packet=login/serverbound/AllCharacterListSelect version=gms_v79 ida=0x5ccc1f
func TestAllCharacterListSelectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AllCharacterListSelect{
				characterId: 12345,
				worldId:     world.Id(2),
				mac:         "AA:BB:CC",
				hwid:        "HWID",
			}
			output := AllCharacterListSelect{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
			if output.Mac() != input.Mac() {
				t.Errorf("mac: got %v, want %v", output.Mac(), input.Mac())
			}
			if output.Hwid() != input.Hwid() {
				t.Errorf("hwid: got %v, want %v", output.Hwid(), input.Hwid())
			}
		})
	}
}
