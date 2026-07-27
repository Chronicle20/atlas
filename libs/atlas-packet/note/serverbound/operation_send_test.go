package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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

// TestOperationSendByteOutputAllVersions is the tier-1 byte-golden fixture
// promoting note/serverbound/NoteOperationSend across all nine coverage-matrix
// versions (task-137 notesend-verify2). Every version's client-side writer is
// CCashShop::OnCashItemResLoadGiftDone — except gms_v48, whose combined
// locker-load+gift-forward handler is CCashShop::OnCashItemResLoadLockerDone
// (see the codec's sendHasGiftDetails doc and packet-audit:fname wrinkle
// resolved via the note/serverbound/CCashShop::OnCashItemResLoadLockerDone
// candidatesFromFName case in tools/packet-audit/cmd/run.go).
//
// Confirmed IDA read/write order per version (decompile; disasm used where
// Hex-Rays truncated the pseudocode to the load-only prefix and hid the
// per-entry gift-forward loop — gms_v61/v72/v79/v84):
//
//	gms_v48  CCashShop::OnCashItemResLoadLockerDone@0x453a91: Encode1(mode=0)@0x453c3d
//	         + EncodeStr(toName) + EncodeStr(message) + Encode1(giftFlag=1)@0x453c84 — NO
//	         giftIndex/giftSN (predates those fields, sendHasGiftDetails()==false).
//	gms_v61  CCashShop::OnCashItemResLoadGiftDone@0x461758 (disasm): Encode1(mode=0)@0x4618a6
//	         + EncodeStr(toName) + EncodeStr(message) + Encode1(giftFlag)@0x4618ed
//	         + Encode4(giftIndex)@0x4618f6 + EncodeBuffer(giftSN,8)@0x461909.
//	gms_v72  CCashShop::OnCashItemResLoadGiftDone@0x47122e (disasm): same shape,
//	         Encode1(mode=0)@0x47136b .. EncodeBuffer(giftSN,8)@0x4713ce.
//	gms_v79  CCashShop::OnCashItemResLoadGiftDone@0x47252c (disasm): same shape,
//	         Encode1(mode=0)@0x472669 .. EncodeBuffer(giftSN,8)@0x4726cc.
//	gms_v83  CCashShop::OnCashItemResLoadGiftDone@0x47959e: same shape,
//	         Encode1(mode=0)@0x4796db .. EncodeBuffer(giftSN,8)@0x47973e.
//	gms_v84  CCashShop::OnCashItemResLoadGiftDone@0x47c73c (disasm): same shape,
//	         Encode1(mode=0)@0x47c879 .. EncodeBuffer(giftSN,8)@0x47c8dc.
//	gms_v87  CCashShop::OnCashItemResLoadGiftDone@0x484cc5: same shape,
//	         Encode1(mode=0)@0x484e02 .. EncodeBuffer(giftSN,8)@0x484e65.
//	gms_v95  CCashShop::OnCashItemResLoadGiftDone@0x496520: same shape (inlined
//	         byte writes into a growable buffer rather than discrete Encode* calls,
//	         same field order: flag@0x4967d3, index@0x496823, sn@0x496874/0x496879).
//	jms_v185 CCashShop::OnCashItemResLoadGiftDone@0x48bdc8: same shape,
//	         Encode1(mode=0)@0x48bf4c .. EncodeBuffer(giftSN,8)@0x48bf68.
//
// The leading Encode1(mode=0) byte is NOT part of OperationSend's own wire —
// it is the shared NOTE_ACTION sub-op discriminator read by the note/serverbound
// Operation wrapper (operation.go) before dispatching to OperationSend; every
// version's byte count below starts after that byte.
//
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v48 ida=0x453a91
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v61 ida=0x461758
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v72 ida=0x47122e
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v79 ida=0x47252c
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v83 ida=0x47959e
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v84 ida=0x47c73c
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v87 ida=0x484cc5
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=gms_v95 ida=0x496520
// packet-audit:verify packet=note/serverbound/NoteOperationSend version=jms_v185 ida=0x48bdc8
func TestOperationSendByteOutputAllVersions(t *testing.T) {
	cases := []struct {
		variant     pt.TenantVariant
		giftDetails bool // false only for gms_v48 (sendHasGiftDetails)
	}{
		{pt.Variants[7], false}, // GMS v48  — OnCashItemResLoadLockerDone@0x453a91
		{pt.Variants[8], true},  // GMS v61  — OnCashItemResLoadGiftDone@0x461758
		{pt.Variants[9], true},  // GMS v72  — OnCashItemResLoadGiftDone@0x47122e
		{pt.Variants[10], true}, // GMS v79  — OnCashItemResLoadGiftDone@0x47252c
		{pt.Variants[1], true},  // GMS v83  — OnCashItemResLoadGiftDone@0x47959e
		{pt.Variants[5], true},  // GMS v84  — OnCashItemResLoadGiftDone@0x47c73c
		{pt.Variants[2], true},  // GMS v87  — OnCashItemResLoadGiftDone@0x484cc5
		{pt.Variants[3], true},  // GMS v95  — OnCashItemResLoadGiftDone@0x496520
		{pt.Variants[4], true},  // JMS v185 — OnCashItemResLoadGiftDone@0x48bdc8
	}

	baseWant := []byte{
		0x03, 0x00, // toName length = 3
		'B', 'o', 'b', // toName = "Bob"
		0x02, 0x00, // message length = 2
		'h', 'i', // message = "hi"
		0x01, // giftFlag = 1
	}
	giftDetailsWant := []byte{
		0x02, 0x00, 0x00, 0x00, // giftIndex = 2
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // giftSN = 0x0102030405060708 (LE)
	}

	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			input := OperationSend{
				toName:    "Bob",
				message:   "hi",
				giftFlag:  1,
				giftIndex: 2,
				giftSN:    0x0102030405060708,
			}
			want := append([]byte{}, baseWant...)
			if tc.giftDetails {
				want = append(want, giftDetailsWant...)
			}
			got := pt.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(got, want) {
				t.Fatalf("NoteOperationSend byte-golden mismatch\n got: % x\nwant: % x", got, want)
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
			if tc.giftDetails {
				if output.GiftIndex() != input.GiftIndex() {
					t.Errorf("giftIndex: got %v, want %v", output.GiftIndex(), input.GiftIndex())
				}
				if output.GiftSN() != input.GiftSN() {
					t.Errorf("giftSN: got %v, want %v", output.GiftSN(), input.GiftSN())
				}
			} else {
				if output.GiftIndex() != 0 {
					t.Errorf("giftIndex: got %v, want 0 (gms_v48 has no giftIndex field)", output.GiftIndex())
				}
				if output.GiftSN() != 0 {
					t.Errorf("giftSN: got %v, want 0 (gms_v48 has no giftSN field)", output.GiftSN())
				}
			}
		})
	}
}
