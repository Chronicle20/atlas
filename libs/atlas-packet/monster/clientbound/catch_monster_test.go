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
	input := NewCatchMonster(0x07654321, 0x42, 0x01)

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

// TestCatchMonsterBytesV79 pins the exact wire bytes against the v79 client read
// order. CATCH_MONSTER (op 229) is a per-mob OnMobPacket case: CMobPool::OnMobPacket
// @0x646d46 reads a uniqueId (Decode4 @0x646d50) -> GetMob, THEN dispatches to
// CMob::OnCatchEffect @0x63c6a9 (GMS_v79_1_DEVM.exe, port 13340) which reads:
//
//	Decode1 @0x63c6b0 — result byte (-> sub_637DEC / ShowCatchEffect); no success byte.
//
// So the v79 wire is [uniqueId int32][result byte]. The leading uniqueId is the
// universal CMobPool::OnMobPacket prefix (see legacyMobPoolPrefix); it is written
// for the pre-v83 legacy range and gated off for v83+ (frozen per campaign).
//
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonster version=gms_v79 ida=0x63c6a9
func TestCatchMonsterBytesV79(t *testing.T) {
	input := NewCatchMonster(0x07654321, 0x42, 0x01)
	ctx := pt.CreateContext("GMS", 79, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4 @0x646d50)
		0x42, // result byte (Decode1 @0x63c6b0)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 catchMonster bytes:\n got % x\nwant % x", got, want)
	}
}
