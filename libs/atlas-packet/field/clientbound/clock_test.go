package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v79 ida=0x5215de
// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v83 ida=0x5361bd
// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v87 ida=0x55DA5F
// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v95 ida=0x531510
// packet-audit:verify packet=field/clientbound/FieldClock version=jms_v185 ida=0x56e849
// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v84 ida=0x5424c1
// TestClockByteOutputV79 pins the gms_v79 CLOCK (op 0x8B) clientbound wire. IDA:
// CField::OnClock @0x5215de (GMS_v79_1_DEVM.exe, named this session — formerly
// sub_5215DE, dispatched via CField::OnPacket case 139 vtable+0x24). Decode1(type)
// @0x5215f5 then per-type: EventClock(0)=Decode4 @0x5217ee; TownClock(1)=3x Decode1
// @0x5217bf/cc/ce; TimerClock(2)=Decode4 @0x52179c; EventTimerClock(3)=Decode1(flag)
// @0x52165a + Decode4 @0x5216e7. Identical read order to v83 CField::OnClock.
func TestClockByteOutputV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)

	event := NewEventClock(300)
	if got := test.Encode(t, ctx, event.Encode, nil); !bytes.Equal(got, []byte{0x00, 0x2C, 0x01, 0x00, 0x00}) {
		t.Errorf("v79 event clock: got %v", got)
	}
	town := NewTownClock(14, 30, 45)
	if got := test.Encode(t, ctx, town.Encode, nil); !bytes.Equal(got, []byte{0x01, 0x0E, 0x1E, 0x2D}) {
		t.Errorf("v79 town clock: got %v", got)
	}
	timer := NewTimerClock(600)
	if got := test.Encode(t, ctx, timer.Encode, nil); !bytes.Equal(got, []byte{0x02, 0x58, 0x02, 0x00, 0x00}) {
		t.Errorf("v79 timer clock: got %v", got)
	}
	eventTimer := NewEventTimerClock(120)
	if got := test.Encode(t, ctx, eventTimer.Encode, nil); !bytes.Equal(got, []byte{0x03, 0x01, 0x78, 0x00, 0x00, 0x00}) {
		t.Errorf("v79 event timer clock: got %v", got)
	}
}

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
