package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestDamage(t *testing.T) {
	in := NewDamage(1000001, 1234, 9300018)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}

// summonDamageHandleV83Body is the Cosmic-baseline (pre-95) serverbound summon
// DAMAGE layout: oid + attackIdx(0-fill) + damage + monsterIdFrom.
//
//	oid=1000001=0x000F4241, attackIdx=0, damage=1234=0x000004D2,
//	monsterIdFrom=9300018=0x008DE832.
var summonDamageHandleV83Body = []byte{
	0x41, 0x42, 0x0F, 0x00, // oid
	0x00,                   // attackIdx (0-fill)
	0xD2, 0x04, 0x00, 0x00, // damage
	0x32, 0xE8, 0x8D, 0x00, // monsterIdFrom
}

// TestDamageBytesV83 pins the classic (pre-95) serverbound DAMAGE layout — no
// trailing dir byte (summon-packet-delta.md §3.5).
func TestDamageBytesV83(t *testing.T) {
	in := NewDamage(1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 83, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, summonDamageHandleV83Body) {
		t.Fatalf("v83 bytes = % X, want % X", got, summonDamageHandleV83Body)
	}
}

// TestDamageBytesV95 pins the v95+ DELTA (gated >= 95, GMS only): the v83 body
// plus a trailing dir<0 flag byte = 0, matching the v95 client send site
// CSummoned::SetDamaged@0x74b730 (Encode1 nDir<0 @0x74bbed; v87's
// SetDamaged@0x7f879a is byte-identical).
// packet-audit:verify packet=summon/serverbound/SummonDamageHandle version=gms_v95 ida=0x74b730
func TestDamageBytesV95(t *testing.T) {
	in := NewDamage(1000001, 1234, 9300018)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)

	want := append(append([]byte{}, summonDamageHandleV83Body...), 0x00) // + trailing dir flag = 0
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 bytes = % X, want % X", got, want)
	}
	if len(got) != len(summonDamageHandleV83Body)+1 {
		t.Fatalf("v95 len = %d, want v83 len + 1 = %d", len(got), len(summonDamageHandleV83Body)+1)
	}
}
