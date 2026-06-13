package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/BuffCancelForeign version=gms_v83 ida=0x983921
// packet-audit:verify packet=character/clientbound/BuffCancelForeign version=gms_v87 ida=0xa093ab
// packet-audit:verify packet=character/clientbound/BuffCancelForeign version=gms_v95 ida=0x953e40
func TestBuffCancelRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffCancel(*cts)
			output := BuffCancel{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestBuffCancelForeignRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffCancelForeign(99999, *cts)
			output := BuffCancelForeign{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != 99999 {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), 99999)
			}
		})
	}
}
