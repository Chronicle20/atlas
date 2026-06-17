package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldWeddingTalk version=gms_v83 ida=0x58153d
// packet-audit:verify packet=field/serverbound/FieldWeddingTalk version=gms_v84 ida=0x5911e6
// packet-audit:verify packet=field/serverbound/FieldWeddingTalk version=gms_v87 ida=0x5b012e
// packet-audit:verify packet=field/serverbound/FieldWeddingTalk version=gms_v95 ida=0x5640f0
func TestWeddingTalkGolden(t *testing.T) {
	input := NewWeddingTalk()
	ctx := pt.CreateContext("GMS", 83, 1)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestWeddingTalkRoundTrip(t *testing.T) {
	input := NewWeddingTalk()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := WeddingTalk{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
