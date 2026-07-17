package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestWorldMessageSimpleByteOutputV79 pins the gms_v79 SERVERMESSAGE (op 0x041)
// clientbound wire for the simple Notice/PopUp/Megaphone/PinkText modes.
//
// IDA-verified client decode (GMS_v79_1_DEVM.exe, port 13340) —
// CWvsContext::OnBroadcastMsg @0x96c94f:
//
//	v3 = CInPacket::Decode1   @0x96c96f → mode byte (v129).
//	(mode != 4, so the mode-4 "hasMessage" prefix branch @0x96c9a9 is skipped)
//	CInPacket::DecodeStr      @0x96c9bd → message string.
//	The trailing switch(mode) at LABEL_23 (@0x96cb83) reads NO further wire bytes
//	for the simple modes (case 0 Notice / case 1 PopUp / case 2 Megaphone /
//	case 5 PinkText all build a UI element from the already-decoded message).
//
// So the simple-mode wire is exactly Decode1(mode) + DecodeStr(message), which
// the atlas WorldMessageSimple.Encode writes (WriteByte(mode) +
// WriteAsciiString(message)). WriteAsciiString = uint16-LE length + ASCII bytes
// (admin_chat golden "hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSimple version=gms_v79 ida=0x96c94f
func TestWorldMessageSimpleByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)

	// Notice mode (0): Decode1(mode) + DecodeStr("hi").
	// 0x00 | 0x02 0x00 'h' 'i'
	input := WorldMessageSimple{mode: 0, message: "hi"}
	expected := []byte{0x00, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 servermessage golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWorldMessageSimpleByteOutputV72 pins the gms_v72 SERVERMESSAGE clientbound
// wire for the simple Notice/PopUp/Megaphone/PinkText modes.
//
// IDA-verified client decode (GMS_v72.1_U_DEVM.exe, port 13339) —
// CWvsContext::OnBroadcastMsg @0x91aaac:
//
//	v3 = CInPacket::Decode1   @0x91aacc → mode byte (v122).
//	(mode != 4, so the mode-4 "hasMessage" prefix Decode1 @0x91ab06 is skipped)
//	CInPacket::DecodeStr      @0x91ab1a → message string.
//	The first switch(mode) @0x91ab44 reads extra fields only for cases 3/8/9/10/11;
//	modes 0/1/2/5 fall through with NO extra wire read. The LABEL_23 switch (case 0
//	Notice body @0x91ace7 / case 1 PopUp / case 2 Megaphone / case 5 PinkText
//	@0x91b798) builds a UI element from the already-decoded message, reading NO
//	further wire bytes.
//
// So the simple-mode wire is exactly Decode1(mode) + DecodeStr(message) — same
// layout as v79. WriteAsciiString = uint16-LE length + ASCII ("hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSimple version=gms_v72 ida=0x91aaac
func TestWorldMessageSimpleByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)

	// Notice mode (0): Decode1(mode) + DecodeStr("hi").
	// 0x00 | 0x02 0x00 'h' 'i'
	input := WorldMessageSimple{mode: 0, message: "hi"}
	expected := []byte{0x00, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 servermessage golden mismatch: got %v want %v", actual, expected)
	}
}

func TestWorldMessageSimpleRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageSimple{mode: 0, message: "Server notice"}
			output := WorldMessageSimple{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

func TestWorldMessageTopScrollRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageTopScroll{mode: 4, message: "Scrolling message"}
			output := WorldMessageTopScroll{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

func TestWorldMessageSuperMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageSuperMegaphone{mode: 3, message: "Super mega!", channelId: 5, whispersOn: true}
			output := WorldMessageSuperMegaphone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if output.WhispersOn() != input.WhispersOn() {
				t.Errorf("whispersOn: got %v, want %v", output.WhispersOn(), input.WhispersOn())
			}
		})
	}
}

func TestWorldMessageBlueTextRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageBlueText{mode: 6, message: "Blue text msg", itemId: 2000000}
			output := WorldMessageBlueText{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

func TestWorldMessageMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewWorldMessageMegaphone(2, "Megaphone message")
			output := WorldMessageMegaphone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

func TestWorldMessageItemMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			item := model.NewAsset(true, 5, 4001126, time.Time{}).SetStackableInfo(30, 0, 0)
			input := NewWorldMessageItemMegaphone(8, "selling stuff", 2, true, &item)
			output := WorldMessageItemMegaphone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() || output.ChannelId() != input.ChannelId() || output.WhispersOn() != input.WhispersOn() {
				t.Errorf("scalar fields did not round-trip")
			}
			if !output.HasItem() {
				t.Errorf("hasItem: got false, want true")
			}
			// no-item variant
			input2 := NewWorldMessageItemMegaphone(8, "no item", 2, false, nil)
			output2 := WorldMessageItemMegaphone{}
			pt.RoundTrip(t, ctx, input2.Encode, output2.Decode, nil)
			if output2.HasItem() {
				t.Errorf("hasItem: got true, want false")
			}
		})
	}
}

func TestWorldMessageYellowMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageYellowMegaphone{mode: 9, message: "Yellow mega!", channelId: 3}
			output := WorldMessageYellowMegaphone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
		})
	}
}

func TestWorldMessageMultiMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageMultiMegaphone{
				mode:       10,
				messages:   []string{"Line one", "Line two", "Line three"},
				channelId:  1,
				whispersOn: true,
			}
			output := WorldMessageMultiMegaphone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if len(output.Messages()) != len(input.Messages()) {
				t.Fatalf("messages length: got %v, want %v", len(output.Messages()), len(input.Messages()))
			}
			for i, msg := range output.Messages() {
				if msg != input.Messages()[i] {
					t.Errorf("messages[%d]: got %v, want %v", i, msg, input.Messages()[i])
				}
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if output.WhispersOn() != input.WhispersOn() {
				t.Errorf("whispersOn: got %v, want %v", output.WhispersOn(), input.WhispersOn())
			}
		})
	}
}

func TestWorldMessageUnknown3RoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageUnknown3{mode: 3, message: "Unknown3 msg", operator: 12345}
			output := WorldMessageUnknown3{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.Operator() != input.Operator() {
				t.Errorf("operator: got %v, want %v", output.Operator(), input.Operator())
			}
		})
	}
}

func TestWorldMessageUnknown7RoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageUnknown7{mode: 7, message: "Unknown7 msg"}
			output := WorldMessageUnknown7{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

func TestWorldMessageUnknown8RoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageUnknown8{mode: 8, message: "Unknown8 msg", channelId: 4, whispersOn: true}
			output := WorldMessageUnknown8{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if output.WhispersOn() != input.WhispersOn() {
				t.Errorf("whispersOn: got %v, want %v", output.WhispersOn(), input.WhispersOn())
			}
		})
	}
}

func TestWorldMessageGachaponRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WorldMessageGachapon{mode: 11, message: "PlayerName", townName: "Henesys", itemId: 1002000}
			output := WorldMessageGachapon{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.TownName() != input.TownName() {
				t.Errorf("townName: got %v, want %v", output.TownName(), input.TownName())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}
