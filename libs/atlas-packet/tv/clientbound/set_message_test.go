package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func testTvAvatar() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002, 6: 1060002, 7: 1072001}
	masked := map[slot.Position]uint32{}
	pets := map[int8]uint32{}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, masked, pets)
}

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342) —
// CMapleTVMan::OnSetMessage@0x6371c1:
//
//	a2_3   = Decode1(a2);              // flag (bit1 = receiverLook present)
//	this[988] = Decode1(a2);           // messageType
//	AvatarLook::Decode(senderLook, a2);
//	DecodeStr -> senderName
//	DecodeStr -> receiverName
//	DecodeStr x5 -> lines[0..4]
//	this[243] = Decode4(a2);           // totalWaitSeconds
//	if (flag & 2) AvatarLook::Decode(receiverLook, a2);
//	Matches TvSetMessage.Encode exactly: flag, messageType, senderLook,
//	senderName, receiverName, 5 lines, totalWaitSeconds, [receiverLook].
//
// packet-audit:verify packet=tv/clientbound/TvTvSetMessage version=gms_v83 ida=0x6371c1
func TestTvSetMessageRoundTripNoReceiver(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			senderLook := testTvAvatar()
			lines := [5]string{"line one", "line two", "line three", "line four", "line five"}
			input := NewTvSetMessage(0, senderLook, "SenderName", "", lines, 120, nil)
			output := TvSetMessage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Flag() != 1 {
				t.Errorf("flag: got %v, want 1", output.Flag())
			}
			if output.MessageType() != input.MessageType() {
				t.Errorf("messageType: got %v, want %v", output.MessageType(), input.MessageType())
			}
			if output.SenderName() != input.SenderName() {
				t.Errorf("senderName: got %v, want %v", output.SenderName(), input.SenderName())
			}
			if output.ReceiverName() != "" {
				t.Errorf("receiverName: got %q, want empty", output.ReceiverName())
			}
			if output.Lines() != input.Lines() {
				t.Errorf("lines: got %v, want %v", output.Lines(), input.Lines())
			}
			if output.TotalWaitSeconds() != input.TotalWaitSeconds() {
				t.Errorf("totalWaitSeconds: got %v, want %v", output.TotalWaitSeconds(), input.TotalWaitSeconds())
			}
			if output.ReceiverLook() != nil {
				t.Errorf("receiverLook: got non-nil, want nil")
			}
			if len(output.SenderLook().Equipment()) != len(senderLook.Equipment()) {
				t.Errorf("senderLook equipment count: got %v, want %v", len(output.SenderLook().Equipment()), len(senderLook.Equipment()))
			}
		})
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CMapleTVMan::OnSetMessage@0x60f870:
//
//	nFlag = Decode1(iPacket)                        // flag (bit1=receiverLook)
//	this->m_nMessageType = Decode1(iPacket)          // messageType
//	AvatarLook::Decode(al1, iPacket) -> m_alSender = al1
//	DecodeStr -> sSender -> DecodeStr -> sReceiver
//	DecodeStr x5 -> sMsg1..sMsg5
//	this->m_nTotalWaitTime = Decode4(iPacket)
//	if (nFlag & 2) { AvatarLook::Decode(al2, iPacket) -> m_alReceiver = al2 }
//	Read order matches TvSetMessage.Encode exactly: flag, messageType,
//	senderLook, senderName, receiverName, 5 lines, totalWaitSeconds,
//	[receiverLook] — byte-identical to gms_v83 (confirms "IDA v83≡v95").
//
// packet-audit:verify packet=tv/clientbound/TvTvSetMessage version=gms_v95 ida=0x60f870
func TestTvSetMessageRoundTripV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	senderLook := testTvAvatar()
	receiverLook := testTvAvatar()
	lines := [5]string{"line one", "line two", "line three", "line four", "line five"}
	input := NewTvSetMessage(1, senderLook, "SenderName", "ReceiverName", lines, 300, &receiverLook)
	output := TvSetMessage{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Flag() != 3 {
		t.Errorf("flag: got %v, want 3", output.Flag())
	}
	if output.ReceiverName() != "ReceiverName" {
		t.Errorf("receiverName: got %q, want %q", output.ReceiverName(), "ReceiverName")
	}
	if output.ReceiverLook() == nil {
		t.Fatalf("receiverLook: got nil, want non-nil")
	}
	if output.TotalWaitSeconds() != input.TotalWaitSeconds() {
		t.Errorf("totalWaitSeconds: got %v, want %v", output.TotalWaitSeconds(), input.TotalWaitSeconds())
	}
}

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345) —
// CMapleTVMan::OnSetMessage@0x64c7eb (unnamed sub_64C7EB on this stripped
// IDB):
//
//	nFlag=Decode1(flag), messageType=Decode1, AvatarLook::Decode(senderLook),
//	then SEVEN DecodeStr calls (senderName, receiverName, lines[0..4]),
//	totalWaitSeconds=Decode4, if(flag&2) AvatarLook::Decode(receiverLook).
//	Read order matches TvSetMessage.Encode exactly — byte-identical to
//	gms_v83/v95.
//
// packet-audit:verify packet=tv/clientbound/TvTvSetMessage version=gms_v84 ida=0x64c7eb
func TestTvSetMessageRoundTripV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	senderLook := testTvAvatar()
	receiverLook := testTvAvatar()
	lines := [5]string{"line one", "line two", "line three", "line four", "line five"}
	input := NewTvSetMessage(1, senderLook, "SenderName", "ReceiverName", lines, 300, &receiverLook)
	output := TvSetMessage{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Flag() != 3 {
		t.Errorf("flag: got %v, want 3", output.Flag())
	}
	if output.Lines() != input.Lines() {
		t.Errorf("lines: got %v, want %v", output.Lines(), input.Lines())
	}
	if output.TotalWaitSeconds() != input.TotalWaitSeconds() {
		t.Errorf("totalWaitSeconds: got %v, want %v", output.TotalWaitSeconds(), input.TotalWaitSeconds())
	}
	if output.ReceiverLook() == nil {
		t.Fatalf("receiverLook: got nil, want non-nil")
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343) —
// CMapleTVMan::OnSetMessage@0x670230:
//
//	flag=Decode1, messageType=Decode1, AvatarLook::Decode(senderLook), then
//	SEVEN DecodeStr calls (senderName, receiverName, lines[0..4]),
//	totalWaitSeconds=Decode4, if(flag&2) AvatarLook::Decode(receiverLook).
//	Byte-identical to gms_v83/v84/v95.
//
// packet-audit:verify packet=tv/clientbound/TvTvSetMessage version=gms_v87 ida=0x670230
func TestTvSetMessageRoundTripV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	senderLook := testTvAvatar()
	receiverLook := testTvAvatar()
	lines := [5]string{"line one", "line two", "line three", "line four", "line five"}
	input := NewTvSetMessage(1, senderLook, "SenderName", "ReceiverName", lines, 300, &receiverLook)
	output := TvSetMessage{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Flag() != 3 {
		t.Errorf("flag: got %v, want 3", output.Flag())
	}
	if output.Lines() != input.Lines() {
		t.Errorf("lines: got %v, want %v", output.Lines(), input.Lines())
	}
	if output.TotalWaitSeconds() != input.TotalWaitSeconds() {
		t.Errorf("totalWaitSeconds: got %v, want %v", output.TotalWaitSeconds(), input.TotalWaitSeconds())
	}
	if output.ReceiverLook() == nil {
		t.Fatalf("receiverLook: got nil, want non-nil")
	}
}

// jms_v185 is INTENTIONALLY NOT verified here (no packet-audit:verify marker,
// no fixture) — task-123 phase 20 finding, documented not fabricated:
//
// IDA evidence (jms_v185 MapleStory_dump_SCY.exe, port 13344) —
// CMapleTVMan::OnSetMessage@0x6ab3b9: flag=Decode1, messageType=Decode1,
// AvatarLook::Decode(senderLook), DecodeStr(senderName), DecodeStr
// (receiverName), then only ONE further DecodeStr (not five) into a local
// buffer, totalWaitSeconds=Decode4, [receiverLook]. Immediately after the
// single DecodeStr, the client calls sub_6AB182(loadFlash, buf, &tokens) —
// decompiled at @0x6ab182 — which splits that ONE string into up to 5
// display lines using TWO independent mechanisms: (a) an explicit delimiter
// scan for byte value 13 ('\r', `if (*v8 == 13)`), AND (b) GDI-measured
// text-width auto-wrap (COM IFontDisp/OLE text-metrics calls, threshold
// 125/170 px depending on m_bLoadFlash) for any run that doesn't hit a '\r'.
// So the wire is genuinely ONE combined string, not five separate
// WriteAsciiString calls — a real structural divergence from GMS. Recovering
// the exact intended 5-line split on decode would require replicating the
// client's GDI text-metric word-wrap, which is a rendering concern, not a
// wire-format one, and is NOT reproducible from server-side Go code without
// inventing an approximation. This is a genuine blocker (a design decision on
// what join semantics to send, not a producible IDA/export/route gap) —
// left BLOCKED per VERIFYING_A_PACKET.md rather than fabricating a fixture
// against the wrong (5-separate-strings) shape. See task-20-report.md.
func TestTvSetMessageRoundTripWithReceiver(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			senderLook := testTvAvatar()
			receiverLook := testTvAvatar()
			lines := [5]string{"line one", "line two", "line three", "line four", "line five"}
			input := NewTvSetMessage(1, senderLook, "SenderName", "ReceiverName", lines, 300, &receiverLook)
			output := TvSetMessage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Flag() != 3 {
				t.Errorf("flag: got %v, want 3", output.Flag())
			}
			if output.MessageType() != input.MessageType() {
				t.Errorf("messageType: got %v, want %v", output.MessageType(), input.MessageType())
			}
			if output.ReceiverName() != "ReceiverName" {
				t.Errorf("receiverName: got %q, want %q", output.ReceiverName(), "ReceiverName")
			}
			if output.ReceiverLook() == nil {
				t.Fatalf("receiverLook: got nil, want non-nil")
			}
			if len(output.ReceiverLook().Equipment()) != len(receiverLook.Equipment()) {
				t.Errorf("receiverLook equipment count: got %v, want %v", len(output.ReceiverLook().Equipment()), len(receiverLook.Equipment()))
			}
		})
	}
}
