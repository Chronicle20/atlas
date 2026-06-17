package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOff version=gms_v83 ida=0x53300b
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOff version=gms_v84 ida=0x53f291
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOff version=gms_v87 ida=0x55a824
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOff version=gms_v95 ida=0x535a80
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOff version=jms_v185 ida=0x57026d
func TestFieldObstacleOnOffGolden(t *testing.T) {
	input := NewFieldObstacleOnOff("obst", 0x01)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x04, 0x00, 'o', 'b', 's', 't', 0x01, 0x00, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestFieldObstacleOnOffRoundTrip(t *testing.T) {
	input := NewFieldObstacleOnOff("obst", 0x01)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
