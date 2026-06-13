package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v83 ida=0x5361bd
// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v87 ida=0x55DA5F
// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v95 ida=0x531510
// packet-audit:verify packet=field/clientbound/FieldClock version=jms_v185 ida=0x56e849
func TestEventClock(t *testing.T) {
	input := NewEventClock(300)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestTownClock(t *testing.T) {
	input := NewTownClock(14, 30, 45)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestTimerClock(t *testing.T) {
	input := NewTimerClock(600)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestCakePieEventTimerClock(t *testing.T) {
	input := NewCakePieEventTimerClock(120)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
