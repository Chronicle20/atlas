package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseSongPlayerUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseSongPlayer{soundLengthMs: 123456, updateTimeFirst: true}
			output := *NewItemUseSongPlayer(true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SoundLengthMs() != 123456 {
				t.Errorf("soundLengthMs: got %v, want %v", output.SoundLengthMs(), 123456)
			}
		})
	}
}

func TestItemUseSongPlayerNoUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false}
			output := *NewItemUseSongPlayer(false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SoundLengthMs() != 123456 {
				t.Errorf("soundLengthMs: got %v, want %v", output.SoundLengthMs(), 123456)
			}
			if output.UpdateTime() != 77777 {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), 77777)
			}
		})
	}
}

func TestItemUseSongPlayerWireOrder(t *testing.T) {
	// soundLengthMs first, then the trailing updateTime on the versions that
	// trail it (GMS <= v84). Little-endian int32 each.
	// 0x0001E240 == 123456, 0x00012FD1 == 77777.
	tests := []struct {
		name     string
		input    ItemUseSongPlayer
		expected []byte
	}{
		{
			name:     "updateTimeFirst",
			input:    ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: true},
			expected: []byte{0x40, 0xe2, 0x01, 0x00},
		},
		{
			name:     "updateTimeTrails",
			input:    ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false},
			expected: []byte{0x40, 0xe2, 0x01, 0x00, 0xd1, 0x2f, 0x01, 0x00},
		},
	}

	ctx := pt.CreateContext("GMS", 83, 1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := pt.Encode(t, ctx, tt.input.Encode, nil)
			if !bytes.Equal(b, tt.expected) {
				t.Errorf("bytes: got %x, want %x", b, tt.expected)
			}
		})
	}
}

// Task-9 coverage sweep (task-252): per-version byte fixtures for the
// case-20 (song player / jukebox) arm of CWvsContext::SendConsumeCash-
// ItemUseRequest, IDA-swept across all ten in-scope IDBs. Every version
// below carries a single Encode4 of the WZ sound's IWzSound::Getlength
// result, reached via IWzResMan::GetObjectA -> cast to IWzSound ->
// Getlength -> Encode4 (identical shape to the v83/v95 addresses already
// cited on the struct's doc comment). gms_v48 and gms_v61 are recorded
// n-a below (docs/packets/audits/<version>/_unimplemented.json), not
// verified here — see the coverage doc for the positive-proof evidence.
//
// GMS v72 (GMS_v72.1_U_DEVM.exe.i64, session 99e435d8):
// SendConsumeCashItemUseRequest@0x904fe2, jumptable case-20 arm
// @0x906966, GetObjectA@0x906b4e, Encode4@0x906bba. GMS<=v84 trails
// update_time (cashsb.UpdateTimeFirst == false).
// packet-audit:verify packet=cash/serverbound/CashItemUseSongPlayer version=gms_v72 ida=0x904fe2
func TestItemUseSongPlayerBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false}.Encode(nil, ctx)(nil)
	want := []byte{0x40, 0xe2, 0x01, 0x00, 0xd1, 0x2f, 0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// GMS v79 (GMS_v79_1_DEVM.exe.i64, session 5a1cd4f3):
// SendConsumeCashItemUseRequest@0x95634a, jumptable case-20 arm
// @0x957d8e, GetObjectA@0x957f76, Encode4@0x957fe2. GMS<=v84 trails
// update_time.
// packet-audit:verify packet=cash/serverbound/CashItemUseSongPlayer version=gms_v79 ida=0x95634a
func TestItemUseSongPlayerBytesV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false}.Encode(nil, ctx)(nil)
	want := []byte{0x40, 0xe2, 0x01, 0x00, 0xd1, 0x2f, 0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}

// GMS v83 (MapleStory_dump.exe.i64, session 754107bf):
// SendConsumeCashItemUseRequest@0xa0a63f, jumptable case-20 arm
// @0xa0c1a2, GetObjectA@0xa0c391, sound-length getter sub_644DCF (vtable
// +56)@0xa0c3ed, Encode4@0xa0c3f6. GMS<=v84 trails update_time. Marker
// is on the function-entry address, matching the sibling
// CashItemUsePetNameTag convention (candidatesFromFName resolves every
// sub-arm of this shared sender to the same export entry).
// packet-audit:verify packet=cash/serverbound/CashItemUseSongPlayer version=gms_v83 ida=0xa0a63f
func TestItemUseSongPlayerBytesV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	got := ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false}.Encode(nil, ctx)(nil)
	want := []byte{0x40, 0xe2, 0x01, 0x00, 0xd1, 0x2f, 0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("v83 = % X, want % X", got, want)
	}
}

// GMS v84 (GMS_v84.1_U_DEVM.i64, session 46c2a2eb):
// SendConsumeCashItemUseRequest@0xa54a2f, jumptable case-20 arm
// @0xa5656d, GetObjectA@0xa56755, Encode4@0xa567c1. GMS<=v84 trails
// update_time.
// packet-audit:verify packet=cash/serverbound/CashItemUseSongPlayer version=gms_v84 ida=0xa54a2f
func TestItemUseSongPlayerBytesV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	got := ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false}.Encode(nil, ctx)(nil)
	want := []byte{0x40, 0xe2, 0x01, 0x00, 0xd1, 0x2f, 0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("v84 = % X, want % X", got, want)
	}
}

// GMS v87 (GMSv87_4GB.exe.i64, session c0829805):
// SendConsumeCashItemUseRequest@0xa9fef9, jumptable case-20 arm
// @0xaa1ae1, GetObjectA@0xaa1ccb, Encode4@0xaa1d30. GMS v87+ leads
// update_time in the shared header (cashsb.UpdateTimeFirst == true), so
// this arm's own sub-body is the sound length alone.
// packet-audit:verify packet=cash/serverbound/CashItemUseSongPlayer version=gms_v87 ida=0xa9fef9
func TestItemUseSongPlayerBytesV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	got := ItemUseSongPlayer{soundLengthMs: 123456, updateTimeFirst: true}.Encode(nil, ctx)(nil)
	want := []byte{0x40, 0xe2, 0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("v87 = % X, want % X", got, want)
	}
}

// GMS v92 (GMS_v92_1_DEVM.exe.i64, session 019cd393):
// SendConsumeCashItemUseRequest@0x9bfe10, jumptable case-20 arm
// @0x9c2029 (structural clone of the v95 arm: TransientLayer_Exist gate
// sub_9A47F0 then the same GetObjectA/IWzSound/Getlength chain),
// Encode4@0x9c22c4. GMS v87+ leads update_time.
// packet-audit:verify packet=cash/serverbound/CashItemUseSongPlayer version=gms_v92 ida=0x9bfe10
func TestItemUseSongPlayerBytesV92(t *testing.T) {
	ctx := pt.CreateContext("GMS", 92, 1)
	got := ItemUseSongPlayer{soundLengthMs: 123456, updateTimeFirst: true}.Encode(nil, ctx)(nil)
	want := []byte{0x40, 0xe2, 0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("v92 = % X, want % X", got, want)
	}
}

// GMS v95.0 (GMS_v95.0_U_DEVM.exe.i64, session ecc757f4):
// SendConsumeCashItemUseRequest@0x9eb3e0, jumptable case-20 arm
// @0x9ed51e, GetObjectA@0x9ed75a, cast to IWzSound@0x9ed773,
// Getlength@0x9ed7af, Encode4@0x9ed7b9 (the addresses already cited on
// the struct's doc comment, re-confirmed live this pass). GMS v87+ leads
// update_time.
// packet-audit:verify packet=cash/serverbound/CashItemUseSongPlayer version=gms_v95 ida=0x9eb3e0
func TestItemUseSongPlayerBytesV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	got := ItemUseSongPlayer{soundLengthMs: 123456, updateTimeFirst: true}.Encode(nil, ctx)(nil)
	want := []byte{0x40, 0xe2, 0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 = % X, want % X", got, want)
	}
}

// JMS v185 (MapleStory_dump_SCY.exe.i64, session a977912e):
// SendConsumeCashItemUseRequest@0xaef2f5, jumptable case-20 arm
// @0xaf0b7b, GetObjectA@0xaf0d80, Encode4@0xaf0de5. JMS leads
// update_time in the shared header.
// packet-audit:verify packet=cash/serverbound/CashItemUseSongPlayer version=jms_v185 ida=0xaef2f5
func TestItemUseSongPlayerBytesJMSv185(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	got := ItemUseSongPlayer{soundLengthMs: 123456, updateTimeFirst: true}.Encode(nil, ctx)(nil)
	want := []byte{0x40, 0xe2, 0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("jms_v185 = % X, want % X", got, want)
	}
}
