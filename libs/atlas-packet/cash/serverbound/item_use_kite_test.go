package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseKiteUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseKite{message: "congrats!", updateTimeFirst: true}
			output := *NewItemUseKite(true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

func TestItemUseKiteNoUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseKite{message: "congrats!", updateTime: 99999, updateTimeFirst: false}
			output := *NewItemUseKite(false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}

// TestItemUseKiteBytesTrailingUpdateTime pins the wire shape for every GMS
// build at or below v84 (v48/v61/v72/v79/v83/v84), where
// CWvsContext::SendConsumeCashItemUseRequest appends update_time in the shared
// send tail rather than the header (see ItemUse.UpdateTimeFirst).
func TestItemUseKiteBytesTrailingUpdateTime(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, major := range []uint16{48, 61, 72, 79, 83, 84} {
		in := ItemUseKite{message: "hi", updateTime: 0x01020304, updateTimeFirst: false}
		got := in.Encode(l, pt.CreateContext("GMS", major, 1))(nil)
		want := []byte{
			0x02, 0x00, 'h', 'i', // message    — EncodeStr
			0x04, 0x03, 0x02, 0x01, // updateTime — trailing Encode4
		}
		if !bytes.Equal(got, want) {
			t.Errorf("GMS v%d kite sub-body:\n got % x\nwant % x", major, got, want)
		}
	}
}

// TestItemUseKiteBytesLeadingUpdateTime pins the wire shape for GMS v87+ and
// JMS v185, where update_time is a leading header int32 already consumed by
// the common ItemUse prefix, so the sub-body is the message alone.
func TestItemUseKiteBytesLeadingUpdateTime(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ItemUseKite{message: "hi", updateTime: 0x01020304, updateTimeFirst: true}
	want := []byte{0x02, 0x00, 'h', 'i'}
	for _, c := range []struct {
		name   string
		region string
		major  uint16
	}{
		{"gms_v87", "GMS", 87},
		{"gms_v92", "GMS", 92},
		{"gms_v95", "GMS", 95},
		{"jms_v185", "JMS", 185},
	} {
		got := in.Encode(l, pt.CreateContext(c.region, c.major, 1))(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("%s kite sub-body:\n got % x\nwant % x", c.name, got, want)
		}
	}
}
