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

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342) —
// CWvsContext::OnBroadcastMsg@0xa22785:
//
//	Decode1(mode) -> DecodeStr(message) -> switch(mode) case 3: goto LABEL_18:
//	  a4 = Decode1(v2);   // channelId
//	  a5 = Decode1(v2);   // whispersOn
//	  break;
//	Wire: mode(1) + message(str) + channelId(1) + whispersOn(1). Matches
//	WorldMessageSuperMegaphone.Encode exactly.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSuperMegaphone version=gms_v83 ida=0xa22785
func TestWorldMessageSuperMegaphoneByteOutputV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := WorldMessageSuperMegaphone{mode: 3, message: "hi", channelId: 5, whispersOn: true}
	expected := []byte{0x03, 0x02, 0x00, 0x68, 0x69, 0x05, 0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v83 super megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::OnBroadcastMsg@0xa04160:
//
//	v3 = Decode1(mode) -> (mode!=4, hasMessage-prefix branch skipped) ->
//	DecodeStr(sNotice) -> first switch(v3): case 3 goto LABEL_31; case 8:
//	sSpeakerName=Decode1(channelId), bWhisperIcon=Decode1(whispersOn),
//	if(Decode1()) item=GW_ItemSlotBase::Decode(); case 9: ...; case 10:
//	count=Decode1, [Decode1 msg[1] if>1] [Decode1 msg[2] if>2] goto LABEL_31.
//	LABEL_31: sSpeakerName=Decode1(channelId), bWhisperIcon=Decode1(whispersOn).
//	Mode 2 (Megaphone) has NO case arm in the first switch — falls straight
//	to the display switch with no further reads. Wire is exactly mode(1) +
//	message(str), identical to gms_v83. Confirms v95≡v83 for this arm.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMegaphone version=gms_v95 ida=0xa04160
func TestWorldMessageMegaphoneByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewWorldMessageMegaphone(2, "hi")
	expected := []byte{0x02, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341) —
// CWvsContext::OnBroadcastMsg@0xa04160, case 3: goto LABEL_31 directly (no
// pre-reads) -> LABEL_31: channelId=Decode1, whispersOn=Decode1. Wire is
// mode(1) + message(str) + channelId(1) + whispersOn(1), identical to
// gms_v83.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSuperMegaphone version=gms_v95 ida=0xa04160
func TestWorldMessageSuperMegaphoneByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := WorldMessageSuperMegaphone{mode: 3, message: "hi", channelId: 5, whispersOn: true}
	expected := []byte{0x03, 0x02, 0x00, 0x68, 0x69, 0x05, 0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 super megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341) —
// CWvsContext::OnBroadcastMsg@0xa04160, case 8: sSpeakerName=Decode1
// (channelId), bWhisperIcon=Decode1 (whispersOn), if(Decode1()) [hasItem]
// item=GW_ItemSlotBase::Decode(). Same shape as gms_v83: no slotPos byte,
// hasItem+item-block directly. Wire (no item): mode+message+channelId+
// whispersOn+hasItem(0).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageItemMegaphone version=gms_v95 ida=0xa04160
func TestWorldMessageItemMegaphoneByteOutputV95NoItem(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewWorldMessageItemMegaphone(8, "hi", 2, true, nil)
	expected := []byte{0x08, 0x02, 0x00, 0x68, 0x69, 0x02, 0x01, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 item megaphone (no item) golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341) —
// CWvsContext::OnBroadcastMsg@0xa04160, case 10: sWarn=Decode1(count); if
// count>1 DecodeStr(message[1]); if count>2 DecodeStr(message[2]); goto
// LABEL_31: channelId=Decode1, whispersOn=Decode1. Plain channelId+
// whispersOn trailer, NOT a channel*10+ear+1 formula — same as gms_v83.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMultiMegaphone version=gms_v95 ida=0xa04160
func TestWorldMessageMultiMegaphoneByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := WorldMessageMultiMegaphone{mode: 10, messages: []string{"a", "b", "c"}, channelId: 1, whispersOn: true}
	expected := []byte{
		0x0A,             // mode
		0x01, 0x00, 0x61, // message[0]="a"
		0x03,             // count
		0x01, 0x00, 0x62, // message[1]="b"
		0x01, 0x00, 0x63, // message[2]="c"
		0x01, // channelId
		0x01, // whispersOn
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 multi megaphone golden mismatch: got %v want %v", actual, expected)
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

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342) —
// CWvsContext::OnBroadcastMsg@0xa22785:
//
//	v3 = Decode1(mode);                       // common header
//	(mode != 4, so the mode-4 hasMessage-prefix branch is skipped)
//	DecodeStr(message);                       // common message read
//	switch (v3) { case 3: goto LABEL_18; case 8: ...; case 10: ...; case 11: ...; }
//	// mode 2 (MEGAPHONE) has NO case arm in this first (body) switch — it
//	// falls straight through to the second (display) switch with no further
//	// wire reads. Wire is exactly mode(1) + message(str).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMegaphone version=gms_v83 ida=0xa22785
func TestWorldMessageMegaphoneByteOutputV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewWorldMessageMegaphone(2, "hi")
	expected := []byte{0x02, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v83 megaphone golden mismatch: got %v want %v", actual, expected)
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

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342) —
// CWvsContext::OnBroadcastMsg@0xa22785:
//
//	Decode1(mode) -> DecodeStr(message) -> switch(mode) case 8:
//	  a4 = Decode1(v2);              // channelId
//	  a5 = Decode1(v2);              // whispersOn
//	  if (Decode1(v2)) {             // hasItem bool
//	    item = GW_ItemSlotBase::Decode(v2);   // item block directly — NO
//	  }                                        // slotPos byte before/around it
//	  break;
//	Resolves the slotPos question: the client reads hasItem then, when true,
//	the GW_ItemSlotBase block immediately — no separate slot-position byte.
//	Matches WorldMessageItemMegaphone.Encode exactly (Cosmic-cited shape was
//	already correct for v83).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageItemMegaphone version=gms_v83 ida=0xa22785
func TestWorldMessageItemMegaphoneByteOutputV83NoItem(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewWorldMessageItemMegaphone(8, "hi", 2, true, nil)
	// mode(1)=0x08 message(2+2)=02 00 68 69 channelId(1)=02 whispersOn(1)=01 hasItem(1)=00
	expected := []byte{0x08, 0x02, 0x00, 0x68, 0x69, 0x02, 0x01, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v83 item megaphone (no item) golden mismatch: got %v want %v", actual, expected)
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

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342) —
// CWvsContext::OnBroadcastMsg@0xa22785:
//
//	Decode1(mode) -> DecodeStr(message[0])  // common header+first message
//	switch(mode) case 10:
//	  v128 = Decode1(v2);              // count
//	  if (v128 > 1) DecodeStr(message[1]);
//	  if (v128 > 2) DecodeStr(message[2]);
//	  LABEL_18:
//	  a4 = Decode1(v2);                // channelId
//	  a5 = Decode1(v2);                // whispersOn
//	  break;
//	Resolves the trailing-bytes question: v83 trailing data is a plain
//	channelId(byte) + whispersOn(bool) — NOT a "channel*10+ear+1" formula
//	(that Cosmic citation does not apply to v83). Matches
//	WorldMessageMultiMegaphone.Encode exactly; no struct change needed.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMultiMegaphone version=gms_v83 ida=0xa22785
func TestWorldMessageMultiMegaphoneByteOutputV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := WorldMessageMultiMegaphone{mode: 10, messages: []string{"a", "b", "c"}, channelId: 1, whispersOn: true}
	expected := []byte{
		0x0A,             // mode
		0x01, 0x00, 0x61, // message[0]="a"
		0x03,             // count
		0x01, 0x00, 0x62, // message[1]="b"
		0x01, 0x00, 0x63, // message[2]="c"
		0x01, // channelId
		0x01, // whispersOn
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v83 multi megaphone golden mismatch: got %v want %v", actual, expected)
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

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345) — sub_A6DC97 (the
// client's OnBroadcastMsg for opcode 0x46; unnamed/stripped on this IDB)
// @0xa6dc97:
//
//	v3 = Decode1(mode) -> (mode!=4, hasMessage-prefix branch skipped) ->
//	DecodeStr(message) (v6/sub_418C8E copy) -> mode 2 (MEGAPHONE) has no case
//	arm in the pre-decode dispatch chain (only 3/8/9/10/11/12/13 branch) —
//	falls straight to the LABEL_28 display switch with no further wire reads.
//	Wire is exactly mode(1) + message(str), byte-identical to gms_v83/v95.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMegaphone version=gms_v84 ida=0xa6dc97
func TestWorldMessageMegaphoneByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewWorldMessageMegaphone(2, "hi")
	expected := []byte{0x02, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v84 megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345) — sub_A6DC97
// @0xa6dc97: mode==3 goto LABEL_18 directly (no pre-reads) -> LABEL_18:
// channelId=Decode1, whispersOn=Decode1. Wire: mode(1) + message(str) +
// channelId(1) + whispersOn(1), byte-identical to gms_v83/v95.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSuperMegaphone version=gms_v84 ida=0xa6dc97
func TestWorldMessageSuperMegaphoneByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := WorldMessageSuperMegaphone{mode: 3, message: "hi", channelId: 5, whispersOn: true}
	expected := []byte{0x03, 0x02, 0x00, 0x68, 0x69, 0x05, 0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v84 super megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345) — sub_A6DC97
// @0xa6dc97, mode==8 branch: channelId=Decode1, whispersOn=Decode1,
// if(Decode1()) [hasItem] item=GW_ItemSlotBase::Decode. Same shape as
// gms_v83/v95: no slotPos byte, hasItem+item-block directly. Wire (no item):
// mode+message+channelId+whispersOn+hasItem(0).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageItemMegaphone version=gms_v84 ida=0xa6dc97
func TestWorldMessageItemMegaphoneByteOutputV84NoItem(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewWorldMessageItemMegaphone(8, "hi", 2, true, nil)
	expected := []byte{0x08, 0x02, 0x00, 0x68, 0x69, 0x02, 0x01, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v84 item megaphone (no item) golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345) — sub_A6DC97
// @0xa6dc97, mode==10 branch: v137=Decode1(count); if(count>1) DecodeStr
// message[1]; if(count>2) DecodeStr message[2]; LABEL_18: channelId=Decode1,
// whispersOn=Decode1. Plain channelId+whispersOn trailer, same as
// gms_v83/v95 (not a channel*10+ear+1 formula).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMultiMegaphone version=gms_v84 ida=0xa6dc97
func TestWorldMessageMultiMegaphoneByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := WorldMessageMultiMegaphone{mode: 10, messages: []string{"a", "b", "c"}, channelId: 1, whispersOn: true}
	expected := []byte{
		0x0A,             // mode
		0x01, 0x00, 0x61, // message[0]="a"
		0x03,             // count
		0x01, 0x00, 0x62, // message[1]="b"
		0x01, 0x00, 0x63, // message[2]="c"
		0x01, // channelId
		0x01, // whispersOn
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v84 multi megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343) —
// CWvsContext::OnBroadcastMsg@0xab9fd5: v3=Decode1(mode) -> (mode!=4, prefix
// branch skipped) -> DecodeStr(message) -> switch(v3): mode 2 has no case
// arm (only 3/8/9/10/11/12/13) — falls straight to the LABEL_28 display
// switch, no further reads. Wire: mode(1)+message(str), byte-identical to
// gms_v83/v84/v95.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMegaphone version=gms_v87 ida=0xab9fd5
func TestWorldMessageMegaphoneByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewWorldMessageMegaphone(2, "hi")
	expected := []byte{0x02, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v87 megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343) —
// CWvsContext::OnBroadcastMsg@0xab9fd5: case 3: goto LABEL_18 directly ->
// channelId=Decode1, whispersOn=Decode1. Wire: mode(1)+message(str)+
// channelId(1)+whispersOn(1), byte-identical to gms_v83/v84/v95.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSuperMegaphone version=gms_v87 ida=0xab9fd5
func TestWorldMessageSuperMegaphoneByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := WorldMessageSuperMegaphone{mode: 3, message: "hi", channelId: 5, whispersOn: true}
	expected := []byte{0x03, 0x02, 0x00, 0x68, 0x69, 0x05, 0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v87 super megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343) —
// CWvsContext::OnBroadcastMsg@0xab9fd5, case 8: channelId=Decode1,
// whispersOn=Decode1, if(Decode1()) [hasItem] item=GW_ItemSlotBase::Decode.
// Same shape as gms_v83/v84/v95: no slotPos byte. Wire (no item):
// mode+message+channelId+whispersOn+hasItem(0).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageItemMegaphone version=gms_v87 ida=0xab9fd5
func TestWorldMessageItemMegaphoneByteOutputV87NoItem(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewWorldMessageItemMegaphone(8, "hi", 2, true, nil)
	expected := []byte{0x08, 0x02, 0x00, 0x68, 0x69, 0x02, 0x01, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v87 item megaphone (no item) golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343) —
// CWvsContext::OnBroadcastMsg@0xab9fd5, case 10: v131=Decode1(count);
// if(v131>1) DecodeStr message[1]; if(v131>2) DecodeStr message[2];
// LABEL_18: channelId=Decode1, whispersOn=Decode1. Plain trailer, same as
// gms_v83/v84/v95.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMultiMegaphone version=gms_v87 ida=0xab9fd5
func TestWorldMessageMultiMegaphoneByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := WorldMessageMultiMegaphone{mode: 10, messages: []string{"a", "b", "c"}, channelId: 1, whispersOn: true}
	expected := []byte{
		0x0A,             // mode
		0x01, 0x00, 0x61, // message[0]="a"
		0x03,             // count
		0x01, 0x00, 0x62, // message[1]="b"
		0x01, 0x00, 0x63, // message[2]="c"
		0x01, // channelId
		0x01, // whispersOn
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v87 multi megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (jms_v185 MapleStory_dump_SCY.exe, port 13344) —
// CWvsContext::OnBroadcastMsg@0xb0985b: v3=Decode1(mode) -> (mode!=4, prefix
// branch skipped) -> DecodeStr(message) -> mode 2 does not match the
// (3||11||12), 8, 9, 10, 13, 14 pre-decode arms — falls straight to the
// LABEL_27 display switch with no further reads. Wire: mode(1)+message(str),
// same shape as GMS.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMegaphone version=jms_v185 ida=0xb0985b
func TestWorldMessageMegaphoneByteOutputJms(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := NewWorldMessageMegaphone(2, "hi")
	expected := []byte{0x02, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("jms megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (jms_v185 MapleStory_dump_SCY.exe, port 13344) —
// CWvsContext::OnBroadcastMsg@0xb0985b: `if (v3==3 || v3==11 || v3==12) {
// channelId=Decode1; whispersOn=Decode1; goto LABEL_27}` — mode 3 shares
// this minimal pre-decode with two jms-only modes (11/12, unmapped/
// unenrolled per task-18-report.md). Wire for mode 3:
// mode(1)+message(str)+channelId(1)+whispersOn(1), same shape as GMS.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSuperMegaphone version=jms_v185 ida=0xb0985b
func TestWorldMessageSuperMegaphoneByteOutputJms(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := WorldMessageSuperMegaphone{mode: 3, message: "hi", channelId: 5, whispersOn: true}
	expected := []byte{0x03, 0x02, 0x00, 0x68, 0x69, 0x05, 0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("jms super megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (jms_v185 MapleStory_dump_SCY.exe, port 13344) —
// CWvsContext::OnBroadcastMsg@0xb0985b, mode==8: channelId=Decode1,
// whispersOn=Decode1, if(Decode1()) [hasItem] item=GW_ItemSlotBase::Decode.
// Same shape as GMS: no slotPos byte. Wire (no item):
// mode+message+channelId+whispersOn+hasItem(0).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageItemMegaphone version=jms_v185 ida=0xb0985b
func TestWorldMessageItemMegaphoneByteOutputJmsNoItem(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := NewWorldMessageItemMegaphone(8, "hi", 2, true, nil)
	expected := []byte{0x08, 0x02, 0x00, 0x68, 0x69, 0x02, 0x01, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("jms item megaphone (no item) golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (jms_v185 MapleStory_dump_SCY.exe, port 13344) —
// CWvsContext::OnBroadcastMsg@0xb0985b, mode==10: v163=Decode1(count);
// if(v163>1) DecodeStr message[1]; if(v163>2) DecodeStr message[2];
// channelId=Decode1, whispersOn=Decode1. Plain trailer, same as GMS.
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMultiMegaphone version=jms_v185 ida=0xb0985b
func TestWorldMessageMultiMegaphoneByteOutputJms(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := WorldMessageMultiMegaphone{mode: 10, messages: []string{"a", "b", "c"}, channelId: 1, whispersOn: true}
	expected := []byte{
		0x0A,             // mode
		0x01, 0x00, 0x61, // message[0]="a"
		0x03,             // count
		0x01, 0x00, 0x62, // message[1]="b"
		0x01, 0x00, 0x63, // message[2]="c"
		0x01, // channelId
		0x01, // whispersOn
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("jms multi megaphone golden mismatch: got %v want %v", actual, expected)
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
