package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSpouseChat version=gms_v79 ida=0x51d566
// packet-audit:verify packet=field/clientbound/FieldSpouseChat version=gms_v83 ida=0x532087
// packet-audit:verify packet=field/clientbound/FieldSpouseChat version=gms_v84 ida=0x53e30d
// packet-audit:verify packet=field/clientbound/FieldSpouseChat version=gms_v87 ida=0x55991a
// packet-audit:verify packet=field/clientbound/FieldSpouseChat version=gms_v95 ida=0x5357f0
// jms_v185: VERSION-ABSENT (no CField::OnCoupleMessage row in the jms export) — cell is ⬜.
func TestSpouseChatGolden(t *testing.T) {
	// Flattened union read order (CField::OnCoupleMessage):
	//   Decode1(mode) + DecodeStr(sender) + Decode1(flag) + DecodeStr(chatText)
	//     + Decode1(partnerFlag) + DecodeStr(partnerText)
	input := NewSpouseChat(SpouseChatModeOwn, "lover", 0x01, "hi", 0x02, "yo")
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{
		0x04,                                // mode
		0x05, 0x00, 'l', 'o', 'v', 'e', 'r', // sender (len 5)
		0x01,                 // flag
		0x02, 0x00, 'h', 'i', // chatText (len 2)
		0x02,                 // partnerFlag
		0x02, 0x00, 'y', 'o', // partnerText (len 2)
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestSpouseChatByteOutputV79 pins the gms_v79 SPOUSE_CHAT (op 0x80) clientbound
// wire. IDA: CField::OnCoupleMessage @0x51d566 (GMS_v79_1_DEVM.exe). The flattened
// union read order (address-ordered DecodeX) is mode + sender + flag + chatText
// (mode-5 arm: DecodeStr @0x51d5a0, Decode1 @0x51d5ac, DecodeStr @0x51d5b7) +
// partnerFlag + partnerText (mode-4 arm: Decode1 @0x51d685, DecodeStr @0x51d6e5).
func TestSpouseChatByteOutputV79(t *testing.T) {
	input := NewSpouseChat(SpouseChatModeOwn, "lover", 0x01, "hi", 0x02, "yo")
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{
		0x04,                                // mode @0x51d58a
		0x05, 0x00, 'l', 'o', 'v', 'e', 'r', // sender (mode-5 DecodeStr @0x51d5a0)
		0x01,                 // flag (mode-5 Decode1 @0x51d5ac)
		0x02, 0x00, 'h', 'i', // chatText (mode-5 DecodeStr @0x51d5b7)
		0x02,                 // partnerFlag (mode-4 Decode1 @0x51d685)
		0x02, 0x00, 'y', 'o', // partnerText (mode-4 DecodeStr @0x51d6e5)
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 spouse chat golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSpouseChatRoundTrip(t *testing.T) {
	input := NewSpouseChat(SpouseChatModeOwn, "lover", 0x01, "hi there", 0x02, "partner reply")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
