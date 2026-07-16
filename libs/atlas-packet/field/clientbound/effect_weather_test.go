package clientbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v83 ida=0x535179
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v87 ida=0x55c953
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v95 ida=0x5468f0
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=jms_v185 ida=0x5723E6
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v84 ida=0x5413ff
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v48 ida=0x4c95f2
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

// TestFieldEffectWeatherByteOutputV48 pins the gms_v48 BLOW_WEATHER (op 0x56 = 86)
// clientbound wire. IDA: CField::OnBlowWeather = sub_4C95F2 @0x4c95f2
// (GMS_v48_1_DEVM.exe) reads Decode4(itemId) @0x4c9604 then, for a weather-type item
// (sub_47742E==18 && itemId>=0), DecodeStr(message) @0x4c9669 — with NO leading bool.
// The leading `!active` bool is a v83+ addition (v61's sub_4ED39C reads the same
// itemId-first shape), so the codec takes the < 61 legacy branch: itemId + optional
// message.
func TestFieldEffectWeatherByteOutputV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)

	itemIdLE := make([]byte, 4)
	binary.LittleEndian.PutUint32(itemIdLE, 5010000)
	msg := "It's raining!"

	// Start: itemId(4, LE) @0x4c9604 + DecodeStr(message) @0x4c9669. No leading bool.
	start := NewFieldEffectWeatherStart(5010000, msg)
	gotStart := start.Encode(l, ctx)(nil)
	wantStart := append([]byte{}, itemIdLE...)
	wantStart = append(wantStart, byte(len(msg)), 0x00)
	wantStart = append(wantStart, []byte(msg)...)
	if !bytes.Equal(gotStart, wantStart) {
		t.Errorf("v48 weather start: got %v want %v", gotStart, wantStart)
	}

	// End: itemId only, no trailing message.
	end := NewFieldEffectWeatherEnd(5010000)
	gotEnd := end.Encode(l, ctx)(nil)
	if !bytes.Equal(gotEnd, itemIdLE) {
		t.Errorf("v48 weather end: got %v want %v", gotEnd, itemIdLE)
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
