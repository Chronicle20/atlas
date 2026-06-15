package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v83 ida=0x52cb7c
// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v84 ida=0x538c80
// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v87 ida=0x553526
// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v95 ida=0x5413e5
func TestSueCharacterGoldenLegacy(t *testing.T) {
	// v83/v84/v87 lead with the accused character id (int32).
	input := NewSueCharacterLegacy(0x01020304, 0x05, "hi")
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x05, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSueCharacterGoldenV95(t *testing.T) {
	// v95 leads with a sub-command string.
	input := NewSueCharacterV95("hi", 0x05, "ho")
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{0x02, 0x00, 0x68, 0x69, 0x05, 0x02, 0x00, 0x68, 0x6f}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSueCharacterRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			var input SueCharacter
			// Mirror the codec's version branch: string-lead from v95 onward
			// (jms is SUE-absent in practice; its branch choice is moot here).
			if v.MajorVersion >= 95 {
				input = NewSueCharacterV95("alice", 0x05, "spamming")
			} else {
				input = NewSueCharacterLegacy(0x01020304, 0x05, "spamming")
			}
			output := SueCharacter{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() || output.SubCommand() != input.SubCommand() ||
				output.Flag() != input.Flag() || output.Reason() != input.Reason() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
