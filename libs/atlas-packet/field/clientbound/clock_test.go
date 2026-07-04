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
// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v72 ida=0x51a522
// TestClockByteOutputV48 pins the gms_v48 CLOCK (op 0x5A = 90) clientbound wire.
// IDA: v48 receives CLOCK at CField::OnPacket @0x4c66f2 case 'Z'(90) @0x4c67f7,
// which dispatches to the CField primary-vtable slot-7 (offset +28) virtual
// CField::OnClock. The primary CField vtable is unsymbolized in this IDB (the
// case 'Z' target is a secondary-base MI vtable-indirect call), so OnClock's body
// is not statically resolvable — cited at the resolvable dispatch entry 0x4c66f2
// (the registry ida.address), mirroring the accepted gms_v61 precedent (0x4e9ea3).
// The OnClock read order is version-invariant (no codec gate): Decode1(clockType)
// then EventClock(0)=Decode4, TownClock(1)=3×Decode1, TimerClock(2)=Decode4,
// EventTimerClock(3)=Decode1(flag)+Decode4 — so the v48 wire is byte-identical to
// the v61/v72-verified goldens.
// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v48 ida=0x4c66f2
func TestClockByteOutputV48(t *testing.T) {
	ctx := test.CreateContext("GMS", 48, 1)
	event := NewEventClock(300)
	if got := test.Encode(t, ctx, event.Encode, nil); !bytes.Equal(got, []byte{0x00, 0x2C, 0x01, 0x00, 0x00}) {
		t.Errorf("v48 event clock: got %v", got)
	}
	town := NewTownClock(14, 30, 45)
	if got := test.Encode(t, ctx, town.Encode, nil); !bytes.Equal(got, []byte{0x01, 0x0E, 0x1E, 0x2D}) {
		t.Errorf("v48 town clock: got %v", got)
	}
	timer := NewTimerClock(600)
	if got := test.Encode(t, ctx, timer.Encode, nil); !bytes.Equal(got, []byte{0x02, 0x58, 0x02, 0x00, 0x00}) {
		t.Errorf("v48 timer clock: got %v", got)
	}
	eventTimer := NewEventTimerClock(120)
	if got := test.Encode(t, ctx, eventTimer.Encode, nil); !bytes.Equal(got, []byte{0x03, 0x01, 0x78, 0x00, 0x00, 0x00}) {
		t.Errorf("v48 event timer clock: got %v", got)
	}
}

// TestClockByteOutputV61 pins the gms_v61 CLOCK (op 0x6E = 110) clientbound wire.
// IDA: v61 receives CLOCK at CField::OnPacket @0x4e9ea3 case 'n'(110), which
// dispatches to the primary-vtable slot-8 virtual CField::OnClock (the same
// vtable+0x20 indirection as the v72 anchor, where OnClock resolved to
// sub_51A522 @0x51a522). The OnClock read order is version-invariant — Decode1
// (clockType) then per-type EventClock(0)=Decode4, TownClock(1)=3×Decode1,
// TimerClock(2)=Decode4, EventTimerClock(3)=Decode1(flag)+Decode4 — and the
// codec carries no version gate, so the v61 wire is byte-identical to the
// v72-verified golden. Cited at the resolvable v61 dispatch entry 0x4e9ea3
// (OnClock itself is vtable-indirect and not statically resolvable this session).
// packet-audit:verify packet=field/clientbound/FieldClock version=gms_v61 ida=0x4e9ea3
func TestClockByteOutputV61(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)
	event := NewEventClock(300)
	if got := test.Encode(t, ctx, event.Encode, nil); !bytes.Equal(got, []byte{0x00, 0x2C, 0x01, 0x00, 0x00}) {
		t.Errorf("v61 event clock: got %v", got)
	}
	town := NewTownClock(14, 30, 45)
	if got := test.Encode(t, ctx, town.Encode, nil); !bytes.Equal(got, []byte{0x01, 0x0E, 0x1E, 0x2D}) {
		t.Errorf("v61 town clock: got %v", got)
	}
	timer := NewTimerClock(600)
	if got := test.Encode(t, ctx, timer.Encode, nil); !bytes.Equal(got, []byte{0x02, 0x58, 0x02, 0x00, 0x00}) {
		t.Errorf("v61 timer clock: got %v", got)
	}
	eventTimer := NewEventTimerClock(120)
	if got := test.Encode(t, ctx, eventTimer.Encode, nil); !bytes.Equal(got, []byte{0x03, 0x01, 0x78, 0x00, 0x00, 0x00}) {
		t.Errorf("v61 event timer clock: got %v", got)
	}
}

// TestClockByteOutputV72 pins the gms_v72 CLOCK (op 0x087) clientbound wire. IDA:
// CField::OnClock = sub_51A522 @0x51a522 (GMS_v72.1_U_DEVM.exe, dispatched via
// CField::OnPacket @0x515879 case 135 -> vtable+0x20; structurally identical to v79
// CField::OnClock, same clock-window field this[107]). Decode1(clockType) @0x51a539
// then per-type: EventClock(0)=Decode4 @0x51a732; TownClock(1)=3x Decode1
// @0x51a703/710/712; TimerClock(2)=Decode4 @0x51a6e0; EventTimerClock(3)=Decode1(flag)
// @0x51a59e + Decode4 @0x51a62b. Byte-identical read order to the v79 golden.
func TestClockByteOutputV72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)

	event := NewEventClock(300)
	if got := test.Encode(t, ctx, event.Encode, nil); !bytes.Equal(got, []byte{0x00, 0x2C, 0x01, 0x00, 0x00}) {
		t.Errorf("v72 event clock: got %v", got)
	}
	town := NewTownClock(14, 30, 45)
	if got := test.Encode(t, ctx, town.Encode, nil); !bytes.Equal(got, []byte{0x01, 0x0E, 0x1E, 0x2D}) {
		t.Errorf("v72 town clock: got %v", got)
	}
	timer := NewTimerClock(600)
	if got := test.Encode(t, ctx, timer.Encode, nil); !bytes.Equal(got, []byte{0x02, 0x58, 0x02, 0x00, 0x00}) {
		t.Errorf("v72 timer clock: got %v", got)
	}
	eventTimer := NewEventTimerClock(120)
	if got := test.Encode(t, ctx, eventTimer.Encode, nil); !bytes.Equal(got, []byte{0x03, 0x01, 0x78, 0x00, 0x00, 0x00}) {
		t.Errorf("v72 event timer clock: got %v", got)
	}
}

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
