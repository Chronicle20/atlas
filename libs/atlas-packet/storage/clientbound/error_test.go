package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// ErrorSimple covers the dispatcher notice arms (modes 10/11/12/16/17 + the
// default fallthrough): the dispatcher consumes ONLY the mode byte then shows a
// StringPool notice with NO further wire reads (decompile: each case just sets
// the StringPool id and goes to the Notice call). Body = mode byte only. Mode
// bytes are version-stable across the GMS dispatchers (v83 0x7c8a4c, v84
// 0x7eec1a, v87 0x81c336, v95 0x76a990).
// packet-audit:verify packet=storage/clientbound/StorageErrorSimple version=gms_v83 ida=0x7c8a4c
// packet-audit:verify packet=storage/clientbound/StorageErrorSimple version=gms_v84 ida=0x7eec1a
// packet-audit:verify packet=storage/clientbound/StorageErrorSimple version=gms_v87 ida=0x81c336
// packet-audit:verify packet=storage/clientbound/StorageErrorSimple version=gms_v95 ida=0x76a990
// packet-audit:verify packet=storage/clientbound/StorageErrorSimple version=jms_v185 ida=0x84e5a1
func TestStorageErrorSimple(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewStorageErrorSimple(10)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// Body is the mode byte only — no trailing fields.
			b := input.Encode(l, ctx)(nil)
			if len(b) != 1 || b[0] != 10 {
				t.Fatalf("ErrorSimple body: got %v, want [10]", b)
			}
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// UpdateMeso is the currency-only SetGetItems body (dispatcher case 0x13 =
// mode 19 = UPDATE_MESO): Decode1 slotCount, DecodeBuffer(8) flags (atlas writes
// 2 = currency bit), Decode4 meso (read because flag&2). No tab data because no
// tab bit is set. Read order = SetGetItems, version-stable (dispatcher v83
// 0x7c8a4c, v84 0x7eec1a, v87 0x81c336, v95 0x76a990).
// packet-audit:verify packet=storage/clientbound/StorageUpdateMeso version=gms_v83 ida=0x7c8a4c
// packet-audit:verify packet=storage/clientbound/StorageUpdateMeso version=gms_v84 ida=0x7eec1a
// packet-audit:verify packet=storage/clientbound/StorageUpdateMeso version=gms_v87 ida=0x81c336
// packet-audit:verify packet=storage/clientbound/StorageUpdateMeso version=gms_v95 ida=0x76a990
// packet-audit:verify packet=storage/clientbound/StorageUpdateMeso version=jms_v185 ida=0x84e5a1
func TestStorageUpdateMeso(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewStorageUpdateMeso(19, 24, 5000000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// Body: mode(1) slots(1) flags(8)=2 meso(4) = 14 bytes, no tab data.
			b := input.Encode(l, ctx)(nil)
			if len(b) != 14 {
				t.Fatalf("UpdateMeso length: got %d, want 14", len(b))
			}
			if b[0] != 19 || b[1] != 24 {
				t.Fatalf("mode/slots: got %d/%d, want 19/24", b[0], b[1])
			}
			// flags long = 2 (currency bit only)
			if b[2] != 2 {
				t.Errorf("flags low byte: got %d, want 2", b[2])
			}
			meso := uint32(b[10]) | uint32(b[11])<<8 | uint32(b[12])<<16 | uint32(b[13])<<24
			if meso != 5000000 {
				t.Errorf("meso: got %d, want 5000000", meso)
			}
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// ErrorMessage is the only arm with a non-trivial body: dispatcher case 23
// (v83/v84/v87) / case 24 (v95) reads Decode1 (an enabled flag) and, if true,
// DecodeStr(message). Atlas always writes the flag true followed by the string.
// Read order IDA-confirmed in every GMS dispatcher: v83 0x7c8a4c case 23, v84
// 0x7eec1a case 23, v87 0x81c336 case 23, v95 0x76a990 case 0x18 (24). The mode
// byte is the only per-version difference; the body shape is identical.
// packet-audit:verify packet=storage/clientbound/StorageErrorMessage version=gms_v83 ida=0x7c8a4c
// packet-audit:verify packet=storage/clientbound/StorageErrorMessage version=gms_v84 ida=0x7eec1a
// packet-audit:verify packet=storage/clientbound/StorageErrorMessage version=gms_v87 ida=0x81c336
// packet-audit:verify packet=storage/clientbound/StorageErrorMessage version=gms_v95 ida=0x76a990
func TestStorageErrorMessage(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// Per-version mode byte (storage_operation.yaml): 24 on v95, else 23.
			mode := byte(23)
			if v.MajorVersion >= 95 {
				mode = 24
			}
			input := NewStorageErrorMessage(mode, "Test error message")
			b := input.Encode(l, ctx)(nil)
			// Body: mode(1) flag(1)=1 then the AsciiString (len short + bytes).
			if b[0] != mode {
				t.Fatalf("mode: got %d, want %d", b[0], mode)
			}
			if b[1] != 1 {
				t.Errorf("enabled flag: got %d, want 1", b[1])
			}
			strLen := uint16(b[2]) | uint16(b[3])<<8
			if int(strLen) != len("Test error message") {
				t.Errorf("string length prefix: got %d, want %d", strLen, len("Test error message"))
			}
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
