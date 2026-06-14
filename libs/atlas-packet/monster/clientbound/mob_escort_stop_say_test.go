package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_ESCORT_STOP_SAY present in v95 (case 306) + jms (case 0x112). Absent in
// v83/v84/v87 (no escort family).
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortStopSay version=gms_v95 ida=0x64c500
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortStopSay version=jms_v185 ida=0x6f0090
func TestMobEscortStopSay(t *testing.T) {
	// hasText = true → the conditional string + action branch is taken.
	input := NewMobEscortStopSay(0x000007D0, 0x00000001, false, true, "Halt!", 0x00000003)

	// Golden bytes (v95). CMob::OnEscortStopSay @0x64c500:
	//   Decode4 -> duration int32 LE
	//   Decode4 -> chatBalloon int32 LE
	//   Decode1 -> weather bool
	//   Decode1 -> hasText bool; when set:
	//     DecodeStr -> text (uint16 length prefix + ascii)
	//     Decode4   -> action int32 LE
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0xD0, 0x07, 0x00, 0x00, // duration int32 LE = 2000
		0x01, 0x00, 0x00, 0x00, // chatBalloon int32 LE = 1
		0x00,       // weather bool = false
		0x01,       // hasText bool = true
		0x05, 0x00, // text length uint16 LE = 5
		0x48, 0x61, 0x6C, 0x74, 0x21, // "Halt!"
		0x03, 0x00, 0x00, 0x00, // action int32 LE = 3
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobEscortStopSay layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
