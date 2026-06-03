package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestOperationChatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationChat{updateTime: 0x11223344, message: "Hello world"}
			output := OperationChat{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if (v.Region == "GMS" && v.MajorVersion >= 87) || v.Region == "JMS" {
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			} else if output.UpdateTime() != 0 {
				t.Errorf("updateTime should be absent (0) for %s, got %v", v.Name, output.UpdateTime())
			}
		})
	}
}

// TestOperationChatBytes pins the version gate: v83 sends EncodeStr message only
// (IDA CMiniRoomBaseDlg::CheckAndSendChat@0x65f438 = Encode1 op + EncodeStr); v87
// already prepends Encode4 get_update_time (v87 CheckAndSendChat@0x69973e shows
// the field present), as does v95. JMS v185 also prepends update_time
// (CheckAndSendChat@0x6db3ce). Gate: (GMS && MajorVersion>=87) || JMS.
func TestOperationChatBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationChat{updateTime: 0x11223344, message: "hi"}

	// v83: just the string "hi" = 0200 6869
	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	if got83 != "02006869" {
		t.Errorf("v83 bytes: got %s, want 02006869", got83)
	}

	// v87: leading update_time (LE) then "hi" = 44332211 0200 6869 (matches v95)
	got87 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 87, 1))(nil))
	if got87 != "4433221102006869" {
		t.Errorf("v87 bytes: got %s, want 4433221102006869", got87)
	}

	// v95: leading update_time (LE) then "hi" = 44332211 0200 6869
	got95 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil))
	if got95 != "4433221102006869" {
		t.Errorf("v95 bytes: got %s, want 4433221102006869", got95)
	}

	// JMS v185: leading update_time (LE) then "hi" = 44332211 0200 6869
	gotJMS := hex.EncodeToString(input.Encode(l, pt.CreateContext("JMS", 185, 1))(nil))
	if gotJMS != "4433221102006869" {
		t.Errorf("JMS v185 bytes: got %s, want 4433221102006869", gotJMS)
	}
}
