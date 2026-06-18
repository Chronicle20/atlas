package clientbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestForeignNameChangedByteOutput verifies the byte output of ForeignNameChanged
// across all tenant variants. The wire body is version-independent.
//
// IDA evidence (CUserPool::OnUserRemotePacket reads the characterId via Decode4,
// then dispatches to CUserRemote::OnGuildNameChanged which reads the name):
//   v83 OnGuildNameChanged@0x983a6a: DecodeStr(newGuildName);
//       dispatcher CUserPool::OnUserRemotePacket@0x94b39a Decode4(characterId).
//   v84 OnGuildNameChanged@0x9c3e08: DecodeStr(newGuildName).
//   v87 OnGuildNameChanged@0xa094f4: DecodeStr(newGuildName).
//   v95 OnGuildNameChanged@0x9550b0: DecodeStr(newGuildName).
// All four read orders are byte-identical: characterId(4) + name(2+len).
// name="NewGuildName" -> 2+12=14 bytes; total 4+14 = 18 bytes.
//
// packet-audit:verify packet=guild/clientbound/GuildForeignNameChanged version=gms_v83 ida=0x0
// packet-audit:verify packet=guild/clientbound/GuildForeignNameChanged version=gms_v84 ida=0x0
// packet-audit:verify packet=guild/clientbound/GuildForeignNameChanged version=gms_v87 ida=0x0
// packet-audit:verify packet=guild/clientbound/GuildForeignNameChanged version=gms_v95 ida=0x0
// packet-audit:verify packet=guild/clientbound/GuildForeignNameChanged version=jms_v185 ida=0xa5763e
func TestForeignNameChangedByteOutput(t *testing.T) {
	const name = "NewGuildName"
	input := NewForeignNameChanged(1001, name)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			want := 4 + 2 + len(name) // characterId(4) + name(2+len)
			if len(got) != want {
				t.Fatalf("byte count: got %d, want %d", len(got), want)
			}
			if cid := binary.LittleEndian.Uint32(got[0:4]); cid != 1001 {
				t.Errorf("characterId: got %d, want 1001", cid)
			}
			if nameLen := binary.LittleEndian.Uint16(got[4:6]); int(nameLen) != len(name) {
				t.Errorf("name length: got %d, want %d", nameLen, len(name))
			}
			if string(got[6:6+len(name)]) != name {
				t.Errorf("name: got %q, want %q", string(got[6:6+len(name)]), name)
			}
		})
	}
}

func TestForeignNameChangedRoundTrip(t *testing.T) {
	input := NewForeignNameChanged(1001, "NewGuildName")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ForeignNameChanged{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
