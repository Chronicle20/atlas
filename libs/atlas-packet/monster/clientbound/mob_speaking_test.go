package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_SPEAKING present in all five (dispatcher cases 0xFD/259/0x10D/301/0x10E).
// packet-audit:verify packet=monster/clientbound/MonsterMobSpeaking version=gms_v83 ida=0x6711ea
// packet-audit:verify packet=monster/clientbound/MonsterMobSpeaking version=gms_v84 ida=0x687743
// packet-audit:verify packet=monster/clientbound/MonsterMobSpeaking version=gms_v87 ida=0x6ac31e
// packet-audit:verify packet=monster/clientbound/MonsterMobSpeaking version=gms_v95 ida=0x650000
// packet-audit:verify packet=monster/clientbound/MonsterMobSpeaking version=jms_v185 ida=0x6ee398
func TestMobSpeaking(t *testing.T) {
	input := NewMobSpeaking(0x00000003, 0x0000000A)

	// Golden bytes (v83 baseline). CMob::OnMobSpeaking @0x6711ea:
	//   v3 = Decode4 -> speechType int32 LE (TrySpeaking arg 1)
	//   v4 = Decode4 -> action int32 LE     (TrySpeaking arg 2)
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x03, 0x00, 0x00, 0x00, // speechType int32 LE = 3
		0x0A, 0x00, 0x00, 0x00, // action int32 LE = 10
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobSpeaking layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
