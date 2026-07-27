package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 pet clientbound fixtures. Every pet writer is byte-identical to the verified
// GMS v72 wire (no version gate on any pet codec). Live-verified against
// GMS_v61.1_U_DEVM.exe @port 13338; ownerId + slot are read upstream by
// CUser::OnPetPacket before each leaf dispatch.

// TestPetMovementBytesV61: CPet::OnMove@0x613522 forwards the CInPacket to
// CMovePath::OnMovePacket (raw movement blob); no per-field reads in the leaf.
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v61 ida=0x613522
func TestPetMovementBytesV61(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)
	got := NewPetMovement(0x01020304, 0x05, model.Movement{}).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId (upstream)
		0x05,       // slot (upstream)
		0x00, 0x00, // movement StartX
		0x00, 0x00, // movement StartY
		0x00, // movement element count = 0
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

// TestPetChatBytesV61: CPet::OnAction@0x613543 reads Decode1(nType)@0x613574,
// Decode1(nAction)@0x61357c, DecodeStr(message)@0x613585, Decode1(balloon)@0x613598.
// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v61 ida=0x613543
func TestPetChatBytesV61(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)
	got := NewPetChat(0x01020304, 0x05, 0x06, 0x07, "Hi", true).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId (upstream)
		0x05,       // slot (upstream)
		0x06,       // nType Decode1@0x613574
		0x07,       // nAction Decode1@0x61357c
		0x02, 0x00, // msg length DecodeStr@0x613585
		0x48, 0x69, // "Hi"
		0x01, // balloon Decode1@0x613598
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

// TestPetCommandResponseBytesV61: CPet::OnActionCommand@0x61367f reads
// Decode1(mode)@0x6136b5; for mode 0 then Decode1(animation)@0x6136db,
// Decode1(success)@0x6136f4, Decode1(balloon)@0x61381e.
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v61 ida=0x61367f
func TestPetCommandResponseBytesV61(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)
	got := NewPetCommandResponse(0x01020304, 0x05, 0x07, true, false).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId (upstream)
		0x05, // slot (upstream)
		0x00, // mode Decode1@0x6136b5 (NewPetCommandResponse => mode 0)
		0x07, // animation Decode1@0x6136db
		0x01, // success Decode1@0x6136f4
		0x00, // balloon Decode1@0x61381e
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}

// TestPetExcludeResponseBytesV61: CPet::OnLoadExceptionList@0x614e54 reads
// DecodeBuffer(8)(petId)@0x614e6f, Decode1(count)@0x614ea2, then Decode4(excludeId)
// @0x614ebe per entry.
// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v61 ida=0x614e54
func TestPetExcludeResponseBytesV61(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)
	got := NewPetExcludeResponse(0x01020304, 0x05, 0x0807060504030201, []uint32{0x11223344, 0x55667788}).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId (upstream)
		0x05,                                           // slot (upstream)
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, // petId DecodeBuffer(8)@0x614e6f (LE)
		0x02,                   // count Decode1@0x614ea2
		0x44, 0x33, 0x22, 0x11, // excludeId[0] Decode4@0x614ebe (LE)
		0x88, 0x77, 0x66, 0x55, // excludeId[1] Decode4@0x614ebe (LE)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 = % X, want % X", got, want)
	}
}
