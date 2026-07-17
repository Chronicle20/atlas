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
