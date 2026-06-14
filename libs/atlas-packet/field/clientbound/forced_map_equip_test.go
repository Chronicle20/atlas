package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldForcedMapEquip version=gms_v83 ida=0x531b7b
// packet-audit:verify packet=field/clientbound/FieldForcedMapEquip version=gms_v84 ida=0x53de01
// packet-audit:verify packet=field/clientbound/FieldForcedMapEquip version=gms_v87 ida=0x55941a
// packet-audit:verify packet=field/clientbound/FieldForcedMapEquip version=gms_v95 ida=0x52a7e0
// packet-audit:verify packet=field/clientbound/FieldForcedMapEquip version=jms_v185 ida=0x56effa
func TestForcedMapEquipGolden(t *testing.T) {
	input := NewForcedMapEquip()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestForcedMapEquipRoundTrip(t *testing.T) {
	input := NewForcedMapEquip()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
