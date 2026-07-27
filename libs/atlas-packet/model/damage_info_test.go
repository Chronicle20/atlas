package model

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// The meso-explosion DamageInfo entry (task-150) replaces the standard 2-byte
// delay with a 1-byte damage-line count followed by that many 4-byte damages
// (design §2.1; IDA v83 0x96b3fb: Encode1 count byte, then the damage loop,
// then the mob CRC as usual). hits is unused in this mode. Standard-mode
// entries are covered by the AttackInfo round-trip fixtures.
func TestMesoExplosionDamageInfoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			di := NewMesoExplosionDamageInfo()
			di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{111, 222, 333})

			out := NewMesoExplosionDamageInfo()
			pt.RoundTrip(t, ctx, di.Encode, out.Decode, nil)

			if len(out.Damages()) != 3 {
				t.Fatalf("decoded %d damage lines, want 3 (the count byte must size the array)", len(out.Damages()))
			}
			enc1 := pt.Encode(t, ctx, di.Encode, nil)
			enc2 := pt.Encode(t, ctx, out.Encode, nil)
			if !bytes.Equal(enc1, enc2) {
				t.Errorf("re-encode mismatch:\n got % x\nwant % x", enc2, enc1)
			}
		})
	}
}
