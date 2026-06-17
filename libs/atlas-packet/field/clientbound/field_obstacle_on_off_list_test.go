package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=gms_v83 ida=0x533057
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=gms_v84 ida=0x53f2dd
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=gms_v87 ida=0x55a870
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=gms_v95 ida=0x535b00
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=jms_v185 ida=0x5702b9
func TestFieldObstacleOnOffListGolden(t *testing.T) {
	input := NewFieldObstacleOnOffList([]ObstacleState{NewObstacleState("x", 0x00000005)})
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x78, 0x05, 0x00, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestFieldObstacleOnOffListRoundTrip(t *testing.T) {
	input := NewFieldObstacleOnOffList([]ObstacleState{
		NewObstacleState("x", 0x00000005),
		NewObstacleState("obstacle2", 0x00000000),
	})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
