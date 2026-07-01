package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestPartyMemberHPByteOutput verifies the byte output of MemberHP across all
// tenant variants. IDA CUserRemote::OnReceiveHP@0x953f50 reads Decode4(hp)+Decode4(maxHp);
// characterId is consumed upstream by CUserPool::OnUserRemotePacket (dispatcher-prefix).
// Expected wire: WriteInt(characterId=4) + WriteInt(hp=4) + WriteInt(maxHp=4) = 12 bytes,
// version-independent (no gate in encoder).
// packet-audit:verify packet=party/clientbound/PartyMemberHP version=gms_v95 ida=0x953f50
// packet-audit:verify packet=party/clientbound/PartyMemberHP version=gms_v87 ida=0xa09474
// packet-audit:verify packet=party/clientbound/PartyMemberHP version=gms_v83 ida=0x9839ea
// packet-audit:verify packet=party/clientbound/PartyMemberHP version=jms_v185 ida=0xa575be
// packet-audit:verify packet=party/clientbound/PartyMemberHP version=gms_v84 ida=0x9c3d88
func TestPartyMemberHPByteOutput(t *testing.T) {
	const wantBytes = 12 // characterId(4) + hp(4) + maxHp(4)
	input := NewPartyMemberHP(1234, 5000, 10000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			if len(got) != wantBytes {
				t.Errorf("byte count: got %d, want %d", len(got), wantBytes)
			}
		})
	}
}

// TestPartyMemberHPV79 pins the gms_v79 UPDATE_PARTYMEMBER_HP (op 187) wire.
//
// IDA-verified client decode (GMS_v79_1_DEVM.exe, port 13340) —
// CUserRemote::OnReceiveHP @0x8d9b90:
//
//	Decode4 @0x8d9ba2 → hp    (v3, used as 100*hp/maxHp).
//	Decode4 @0x8d9ba9 → maxHp (v4).
//
// characterId is consumed upstream by CUserPool::OnUserRemotePacket (the
// remote-user dispatcher prefix that resolves `this`), so the full wire is
// WriteInt(characterId) + WriteInt(hp) + WriteInt(maxHp) = 12 bytes.
//
// packet-audit:verify packet=party/clientbound/PartyMemberHP version=gms_v79 ida=0x8d9b90
func TestPartyMemberHPV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	m := NewPartyMemberHP(1234, 5000, 10000)
	want := []byte{
		0xD2, 0x04, 0x00, 0x00, // characterId = 1234 (dispatcher prefix)
		0x88, 0x13, 0x00, 0x00, // Decode4 hp = 5000
		0x10, 0x27, 0x00, 0x00, // Decode4 maxHp = 10000
	}
	if got := m.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
		t.Errorf("v79 PartyMemberHP golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// TestPartyMemberHPV72 pins the gms_v72 UPDATE_PARTYMEMBER_HP wire.
//
// IDA-verified client decode (GMS_v72.1_U_DEVM.exe, port 13339) —
// CUserRemote::OnReceiveHP @0x88cc97:
//
//	Decode4 @0x88cca9 → hp    (v3, used as 100*hp/maxHp).
//	Decode4 @0x88ccb0 → maxHp (v4).
//
// Byte-identical to the verified v79 wire. characterId is consumed upstream by
// CUserPool::OnUserRemotePacket (the remote-user dispatcher prefix that resolves
// `this`), so the full wire is WriteInt(characterId) + WriteInt(hp) +
// WriteInt(maxHp) = 12 bytes.
//
// packet-audit:verify packet=party/clientbound/PartyMemberHP version=gms_v72 ida=0x88cc97
func TestPartyMemberHPV72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)
	m := NewPartyMemberHP(1234, 5000, 10000)
	want := []byte{
		0xD2, 0x04, 0x00, 0x00, // characterId = 1234 (dispatcher prefix)
		0x88, 0x13, 0x00, 0x00, // Decode4 hp = 5000 (@0x88cca9)
		0x10, 0x27, 0x00, 0x00, // Decode4 maxHp = 10000 (@0x88ccb0)
	}
	if got := m.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
		t.Errorf("v72 PartyMemberHP golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

func TestPartyMemberHP(t *testing.T) {
	input := NewPartyMemberHP(1234, 5000, 10000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
