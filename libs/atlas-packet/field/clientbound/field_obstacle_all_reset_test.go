package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldFieldObstacleAllReset version=gms_v83 ida=0x5330b6
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleAllReset version=gms_v84 ida=0x53f33c
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleAllReset version=gms_v87 ida=0x55a8cf
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleAllReset version=gms_v95 ida=0x52c830
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleAllReset version=jms_v185 ida=0x570318
func TestFieldObstacleAllResetGolden(t *testing.T) {
	input := NewFieldObstacleAllReset()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestFieldObstacleAllResetRoundTrip(t *testing.T) {
	input := NewFieldObstacleAllReset()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
