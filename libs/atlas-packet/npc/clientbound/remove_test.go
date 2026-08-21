package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=npc/clientbound/NpcRemove version=gms_v48 ida=0x56d5b9
// packet-audit:verify packet=npc/clientbound/NpcRemove version=gms_v61 ida=0x5efe4b
// packet-audit:verify packet=npc/clientbound/NpcRemove version=gms_v72 ida=0x645f45
// packet-audit:verify packet=npc/clientbound/NpcRemove version=gms_v79 ida=0x668ad6
// packet-audit:verify packet=npc/clientbound/NpcRemove version=gms_v83 ida=0x6d9a25
// packet-audit:verify packet=npc/clientbound/NpcRemove version=gms_v84 ida=0x6f0bc8
// packet-audit:verify packet=npc/clientbound/NpcRemove version=gms_v87 ida=0x71706a
// packet-audit:verify packet=npc/clientbound/NpcRemove version=gms_v92 ida=0x66bd70
// packet-audit:verify packet=npc/clientbound/NpcRemove version=gms_v95 ida=0x6792c0
// packet-audit:verify packet=npc/clientbound/NpcRemove version=jms_v185 ida=0x720724
func TestNpcRemove(t *testing.T) {
	input := NewNpcRemove(100123)

	got := input.Encode(nil, test.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0x1B, 0x87, 0x01, 0x00} // objectId 100123 uint32 LE (0x1871B; brief's "0x9B" leading byte is a transcription typo for "0x1B")
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d bytes, want %d bytes\ngot:  % X\nwant: % X", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d mismatch: got %#02x, want %#02x\ngot:  % X\nwant: % X", i, got[i], want[i], got, want)
		}
	}

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
