package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterCatchMonster version=gms_v83 ida=0x66d6b9
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonster version=gms_v84 ida=0x6839bb
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonster version=gms_v87 ida=0x6a8585
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonster version=gms_v95 ida=0x63cd00
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonster version=jms_v185 ida=0x6e5f77
func TestCatchMonster(t *testing.T) {
	input := NewCatchMonster(0x42, 0x01)

	// Golden bytes (v83 baseline). CMob::OnCatchEffect @0x66d6b9:
	//   v3 = Decode1(a1) -> ShowCatchEffect(this, v3) — single byte, no success.
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x42, // result byte (Decode1 @0x66d6b9)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("CatchMonster v83 layout mismatch\n got % x\nwant % x", got, want)
	}

	// Golden bytes (v95). CMob::OnCatchEffect @0x63cd00:
	//   v3 = Decode1; v4 = Decode1; ShowCatchEffect(this, v3, v4!=0?0x10E:0) — two bytes.
	gotV95 := input.Encode(nil, pt.CreateContext("GMS", 95, 0))(nil)
	wantV95 := []byte{
		0x42, // result  byte (Decode1 @0x63cd00)
		0x01, // success byte (Decode1 @0x63cd00)
	}
	if !bytes.Equal(gotV95, wantV95) {
		t.Fatalf("CatchMonster v95 layout mismatch\n got % x\nwant % x", gotV95, wantV95)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
