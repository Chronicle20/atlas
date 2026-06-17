package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=gms_v83 ida=0x5378ba
// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=gms_v84 ida=0x543bb8
// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=gms_v87 ida=0x55f22f
// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=gms_v95 ida=0x52c870
// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=jms_v185 ida=0x574af3
func TestSetQuestClearGolden(t *testing.T) {
	input := NewSetQuestClear()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestSetQuestClearRoundTrip(t *testing.T) {
	input := NewSetQuestClear()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
