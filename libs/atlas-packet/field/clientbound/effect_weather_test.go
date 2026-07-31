package clientbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v83 ida=0x535179
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v87 ida=0x55c953
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v95 ida=0x5468f0
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=jms_v185 ida=0x5723E6
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v84 ida=0x5413ff
// packet-audit:verify packet=field/clientbound/FieldEffectWeather version=gms_v48 ida=0x4c930a
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

// TestFieldEffectWeatherByteOutputV48 pins the gms_v48 BLOW_WEATHER (op 0x55 = 85)
// clientbound wire. IDA (GMS_v48_1_DEVM.exe): CField::OnPacket @0x4c66f2 dispatches
// case 'U'(85) to ?OnBlowWeather@CField@@ = sub_4C930A @0x4c930a, which reads
// Decode1 @0x4c9328, then Decode4(itemId) @0x4c932e, then — for a weather-type item
// (get_consume_cash_item_type @0x47742e) — DecodeStr(message) @0x4c9558, gated on the
// LEADING BYTE being 0. So v48 carries the leading `!active` bool and takes encodeGMS.
//
// task-188 corrected three things here. The op is 85, not 86: case 'V'(86) is
// ?OnPlayJukeBox@CField@@ = sub_4C95F2 @0x4c95f2, a different packet. The previous
// version of this test cited sub_4C95F2 and the addresses 0x4c9604/0x4c9669, which
// lie past the end of sub_4C930A (it ends at 0x4c95e1) — i.e. the no-leading-bool
// shape was read off the jukebox handler. The template and the op registry carried
// the same one-slot shift and were corrected with it.
func TestFieldEffectWeatherByteOutputV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)

	itemIdLE := make([]byte, 4)
	binary.LittleEndian.PutUint32(itemIdLE, 5010000)
	msg := "It's raining!"

	// Start: Decode1 @0x4c9328 == 0 (so the client goes on to read the message),
	// then itemId(4, LE) @0x4c932e, then DecodeStr(message) @0x4c9558.
	start := NewFieldEffectWeatherStart(5010000, msg)
	gotStart := start.Encode(l, ctx)(nil)
	wantStart := []byte{0x00}
	wantStart = append(wantStart, itemIdLE...)
	wantStart = append(wantStart, byte(len(msg)), 0x00)
	wantStart = append(wantStart, []byte(msg)...)
	if !bytes.Equal(gotStart, wantStart) {
		t.Errorf("v48 weather start: got %v want %v", gotStart, wantStart)
	}

	// End: leading byte 1 (non-zero), so the client takes the branch that reads no
	// message; itemId only follows.
	end := NewFieldEffectWeatherEnd(5010000)
	gotEnd := end.Encode(l, ctx)(nil)
	wantEnd := append([]byte{0x01}, itemIdLE...)
	if !bytes.Equal(gotEnd, wantEnd) {
		t.Errorf("v48 weather end: got %v want %v", gotEnd, wantEnd)
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
