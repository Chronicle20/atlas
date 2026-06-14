package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v83 ida=0x66d997
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v84 ida=0x683c9f
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v87 ida=0x6a886e
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=gms_v95 ida=0x63cd40
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonsterWithItem version=jms_v185 ida=0x6eb148
func TestCatchMonsterWithItem(t *testing.T) {
	input := NewCatchMonsterWithItem(0x00226CA0, 0x03)

	// Golden bytes (v83 baseline). CMob::OnEffectByItem @0x66d997:
	//   v3 = Decode4(a2)  -> itemId int32 LE
	//   v4 = Decode1(a2)  -> result byte
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0xA0, 0x6C, 0x22, 0x00, // itemId int32 LE = 0x00226CA0 (Decode4 @0x66d997)
		0x03, // result byte (Decode1 @0x66d997)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("CatchMonsterWithItem layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
