package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v95 ida=0x496520
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v87 ida=0x484cc5
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v83 ida=0x47959e
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=jms_v185 ida=0x48bdc8
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v84 ida=0x47c73c
func TestOperationSendRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationSend{toName: "Recipient", message: "Hello there!"}
			output := OperationSend{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ToName() != input.ToName() {
				t.Errorf("toName: got %v, want %v", output.ToName(), input.ToName())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

// TestOperationSendGiftFieldsByteOutput proves the full gift-forward wire
// OperationSend now models: toName(str), message(str), giftFlag(byte),
// giftIndex(uint32), giftSN(uint64 LE) — not just the toName/message prefix
// exercised by TestOperationSendRoundTrip above. This is a plain unit test,
// not a matrix-promoting fixture (no packet-audit:verify marker): the gift
// fields are decoded because the client writes them unconditionally, but
// they are out of scope for the note feature (design §2.3) and
// NoteOperationSend's matrix cells stay at their current state in this pass.
func TestOperationSendGiftFieldsByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := OperationSend{
		toName:    "Bob",
		message:   "hi",
		giftFlag:  1,
		giftIndex: 2,
		giftSN:    0x0102030405060708,
	}
	want := []byte{
		0x03, 0x00, // toName length = 3
		'B', 'o', 'b', // toName = "Bob"
		0x02, 0x00, // message length = 2
		'h', 'i', // message = "hi"
		0x01,                   // giftFlag = 1
		0x02, 0x00, 0x00, 0x00, // giftIndex = 2
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // giftSN = 0x0102030405060708 (LE)
	}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Fatalf("NoteOperationSend gift-fields golden mismatch\n got: % x\nwant: % x", got, want)
	}

	output := OperationSend{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.ToName() != input.ToName() {
		t.Errorf("toName: got %v, want %v", output.ToName(), input.ToName())
	}
	if output.Message() != input.Message() {
		t.Errorf("message: got %v, want %v", output.Message(), input.Message())
	}
	if output.GiftFlag() != input.GiftFlag() {
		t.Errorf("giftFlag: got %v, want %v", output.GiftFlag(), input.GiftFlag())
	}
	if output.GiftIndex() != input.GiftIndex() {
		t.Errorf("giftIndex: got %v, want %v", output.GiftIndex(), input.GiftIndex())
	}
	if output.GiftSN() != input.GiftSN() {
		t.Errorf("giftSN: got %v, want %v", output.GiftSN(), input.GiftSN())
	}
}
