package clientbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestFieldEffectWeatherStart(t *testing.T) {
	input := NewFieldEffectWeatherStart(5010000, "It's raining!")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestFieldEffectWeatherEnd(t *testing.T) {
	input := NewFieldEffectWeatherEnd(5010000)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestEffectWeatherJMSBranch(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewFieldEffectWeatherStart(5120000, "Happy holidays")
	// JMS185: itemId(4) first (no leading bool), then message (itemId!=0).
	b := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
	if got := binary.LittleEndian.Uint32(b[0:4]); got != 5120000 {
		t.Errorf("JMS leading itemId = %d, want 5120000 (no leading bool)", got)
	}
	// GMS v83 unchanged: leading bool then itemId.
	g := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if g[0] != 0x00 { // !active == false for a start packet
		t.Errorf("GMS leading byte = 0x%02x, want 0x00", g[0])
	}
	if got := binary.LittleEndian.Uint32(g[1:5]); got != 5120000 {
		t.Errorf("GMS itemId (after bool) = %d, want 5120000", got)
	}
	for _, v := range pt.Variants {
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		pt.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
	}
}
