package clientbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestForeignEmblemChangedByteOutput verifies the byte output of
// ForeignEmblemChanged across all tenant variants. The wire body is
// version-independent.
//
// IDA evidence (CUserPool::OnUserRemotePacket reads the characterId via Decode4,
// then dispatches to CUserRemote::OnGuildMarkChanged):
//   v83 OnGuildMarkChanged@0x983ab5: Decode2(nMarkBg)+Decode1(nMarkBgColor)+Decode2(nMark)+Decode1(nMarkColor);
//       dispatcher CUserPool::OnUserRemotePacket@0x94b39a Decode4(characterId).
//   v84 OnGuildMarkChanged@0x9c3e53: Decode2+Decode1+Decode2+Decode1.
//   v87 OnGuildMarkChanged@0xa0953f: Decode2+Decode1+Decode2+Decode1.
//   v95 OnGuildMarkChanged@0x953fe0: Decode2+Decode1+Decode2+Decode1.
// All four read orders are byte-identical:
//   characterId(4) + logoBackground(2) + logoBackgroundColor(1) + logo(2) + logoColor(1) = 10 bytes.
//
// packet-audit:verify packet=guild/clientbound/GuildForeignEmblemChanged version=gms_v83 ida=0x0
// packet-audit:verify packet=guild/clientbound/GuildForeignEmblemChanged version=gms_v84 ida=0x0
// packet-audit:verify packet=guild/clientbound/GuildForeignEmblemChanged version=gms_v87 ida=0x0
// packet-audit:verify packet=guild/clientbound/GuildForeignEmblemChanged version=gms_v95 ida=0x0
// packet-audit:verify packet=guild/clientbound/GuildForeignEmblemChanged version=jms_v185 ida=0xa57689
func TestForeignEmblemChangedByteOutput(t *testing.T) {
	// logo=3, logoColor=2, logoBackground=5, logoBackgroundColor=4
	input := NewForeignEmblemChanged(1001, 3, 2, 5, 4)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			const want = 4 + 2 + 1 + 2 + 1 // characterId + bg + bgColor + mark + markColor
			if len(got) != want {
				t.Fatalf("byte count: got %d, want %d", len(got), want)
			}
			if cid := binary.LittleEndian.Uint32(got[0:4]); cid != 1001 {
				t.Errorf("characterId: got %d, want 1001", cid)
			}
			if bg := binary.LittleEndian.Uint16(got[4:6]); bg != 5 {
				t.Errorf("logoBackground: got %d, want 5", bg)
			}
			if got[6] != 4 {
				t.Errorf("logoBackgroundColor: got %d, want 4", got[6])
			}
			if mark := binary.LittleEndian.Uint16(got[7:9]); mark != 3 {
				t.Errorf("logo: got %d, want 3", mark)
			}
			if got[9] != 2 {
				t.Errorf("logoColor: got %d, want 2", got[9])
			}
		})
	}
}

func TestForeignEmblemChangedRoundTrip(t *testing.T) {
	input := NewForeignEmblemChanged(1001, 3, 2, 5, 4)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ForeignEmblemChanged{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
