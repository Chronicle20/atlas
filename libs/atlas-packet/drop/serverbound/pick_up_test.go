package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestPickUpByteOutputV79 pins the gms_v79 ITEM_PICKUP (op 0x0C2) serverbound
// wire. IDA-verified send site (GMS_v79_1_DEVM.exe, port 13340) —
// CWvsContext::SendDropPickUpRequest @0x954e9d, send block:
//
//	COutPacket::COutPacket(194)              @0x954efb → opcode 0xC2 (registry).
//	COutPacket::Encode1(get_field()+276)     @0x954f18 → fieldKey byte.
//	COutPacket::Encode4(update_time)         @0x954f26 → updateTime uint32-LE.
//	COutPacket::Encode2(pt->x)               @0x954f37 → x int16-LE.
//	COutPacket::Encode2(pt->y)               @0x954f46 → y int16-LE.
//	COutPacket::Encode4(dwDropID)            @0x954f51 → dropId uint32-LE.
//	(NO trailing Encode4 — v79 sends NO client-crc; the crc Encode4 first
//	 appears at v83 @0xa091d7 and v95 @0x9d5eb9, gated by pickUpHasCRC.)
//
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v79 ida=0x954e9d
func TestPickUpByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	// fieldKey=1, updateTime=0x00000064, x=50, y=60, dropId=12345(0x3039).
	input := PickUp{fieldKey: 1, updateTime: 100, x: 50, y: 60, dropId: 12345, crc: 99}
	expected := []byte{
		0x01,                   // fieldKey
		0x64, 0x00, 0x00, 0x00, // updateTime
		0x32, 0x00, // x = 50
		0x3C, 0x00, // y = 60
		0x39, 0x30, 0x00, 0x00, // dropId = 12345
		// no crc on v79
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 pickup golden mismatch: got %v want %v", actual, expected)
	}
}

// TestPickUpByteOutputV61 pins the gms_v61 ITEM_PICKUP (op 169 / 0xA9) serverbound
// wire. IDA-verified send site (GMS_v61.1_U_DEVM.exe, port 13338) —
// sub_8316B8 @0x8316b8 (UNNAMED in the IDB; structurally the
// CWvsContext::SendDropPickUpRequest twin), send block:
//
//	COutPacket::COutPacket(169)             @0x831705 → opcode 0xA9 (registry op 169).
//	COutPacket::Encode1(*(get_field()+248)) @0x831721 → fieldKey byte.
//	COutPacket::Encode4(updateTime)         @0x83172f → updateTime uint32-LE.
//	COutPacket::Encode2(*a2 = x)            @0x831740 → x int16-LE.
//	COutPacket::Encode2(a2[2] = y)          @0x83174f → y int16-LE.
//	COutPacket::Encode4(a3 = dropId)        @0x83175a → dropId uint32-LE.
//	(NO trailing Encode4 — v61<83 sends NO client-crc; the crc Encode4 first
//	 appears at v83 @0xa091d7, gated by pickUpHasCRC = MajorAtLeast(83).)
//
// Byte-identical field order to the VERIFIED v72/v79 wire (only the opcode
// differs: 169 vs 192/194, already in the registry). The codec is version-gated
// on MajorAtLeast(83) for the crc, so v61 emits the no-crc shape.
//
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v61 ida=0x8316b8
func TestPickUpByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	// fieldKey=1, updateTime=0x00000064, x=50, y=60, dropId=12345(0x3039).
	input := PickUp{fieldKey: 1, updateTime: 100, x: 50, y: 60, dropId: 12345, crc: 99}
	expected := []byte{
		0x01,                   // Encode1 fieldKey @0x831721
		0x64, 0x00, 0x00, 0x00, // Encode4 updateTime @0x83172f
		0x32, 0x00, // Encode2 x = 50 @0x831740
		0x3C, 0x00, // Encode2 y = 60 @0x83174f
		0x39, 0x30, 0x00, 0x00, // Encode4 dropId = 12345 @0x83175a
		// no crc on v61 (<83)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v61 pickup golden mismatch: got %v want %v", actual, expected)
	}
}

// TestPickUpByteOutputV72 pins the gms_v72 ITEM_PICKUP (op 192 / 0xC0) serverbound
// wire. IDA-verified send site (GMS_v72.1_U_DEVM.exe, port 13339) —
// CWvsContext::SendDropPickUpRequest = sub_903B77 @0x903b77, send block:
//
//	COutPacket::COutPacket(192)             @0x903bd5 → opcode 0xC0 (registry op 192).
//	COutPacket::Encode1(*(get_field()+276)) @0x903bf2 → fieldKey byte.
//	COutPacket::Encode4(updateTime)         @0x903c00 → updateTime uint32-LE.
//	COutPacket::Encode2(*a2 = x)            @0x903c11 → x int16-LE.
//	COutPacket::Encode2(a2[2] = y)          @0x903c20 → y int16-LE.
//	COutPacket::Encode4(a3 = dropId)        @0x903c2b → dropId uint32-LE.
//	(NO trailing Encode4 — v72<83 sends NO client-crc, same as v79 legacy.)
//
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v72 ida=0x903b77
func TestPickUpByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	// fieldKey=1, updateTime=0x00000064, x=50, y=60, dropId=12345(0x3039).
	input := PickUp{fieldKey: 1, updateTime: 100, x: 50, y: 60, dropId: 12345, crc: 99}
	expected := []byte{
		0x01,                   // Encode1 fieldKey
		0x64, 0x00, 0x00, 0x00, // Encode4 updateTime
		0x32, 0x00, // Encode2 x = 50
		0x3C, 0x00, // Encode2 y = 60
		0x39, 0x30, 0x00, 0x00, // Encode4 dropId = 12345
		// no crc on v72 (<83)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 pickup golden mismatch: got %v want %v", actual, expected)
	}
}

// TestPickUpByteOutputV48 pins the gms_v48 ITEM_PICKUP (op 142 / 0x8E) serverbound
// wire. IDA-verified send site (GMS_v48_1_DEVM.exe, port 13337) —
// sub_70D987 @0x70d987 (UNNAMED in the IDB; structurally the
// CWvsContext::SendDropPickUpRequest twin), send block after the 30ms throttle
// (sub_4A2518(30,0)):
//
//	COutPacket::COutPacket(142)              @0x70d9c6 → opcode 0x8E (registry op 142).
//	COutPacket::Encode1(*(get_field()+216))  @0x70d9e2 → fieldKey byte.
//	COutPacket::Encode4(exclReqTime)         @0x70d9f0 → updateTime uint32-LE.
//	COutPacket::Encode2(*a2 = x)             @0x70da01 → x int16-LE.
//	COutPacket::Encode2(a2[2] = y)           @0x70da10 → y int16-LE.
//	COutPacket::Encode4(a3 = dropId)         @0x70da1b → dropId uint32-LE.
//	(NO trailing Encode4 — v48<83 sends NO client-crc; the crc Encode4 first
//	 appears at v83 @0xa091d7, gated by pickUpHasCRC = MajorAtLeast(83).)
//
// Byte-identical field order to the VERIFIED v61/v72/v79 wire (only the opcode
// differs: 142 vs 169/192/194, already in the registry). The codec is version-
// gated on MajorAtLeast(83) for the crc, so v48 emits the no-crc shape.
//
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v48 ida=0x70d987
func TestPickUpByteOutputV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	// fieldKey=1, updateTime=0x00000064, x=50, y=60, dropId=12345(0x3039).
	input := PickUp{fieldKey: 1, updateTime: 100, x: 50, y: 60, dropId: 12345, crc: 99}
	expected := []byte{
		0x01,                   // Encode1 fieldKey @0x70d9e2
		0x64, 0x00, 0x00, 0x00, // Encode4 updateTime @0x70d9f0
		0x32, 0x00, // Encode2 x = 50 @0x70da01
		0x3C, 0x00, // Encode2 y = 60 @0x70da10
		0x39, 0x30, 0x00, 0x00, // Encode4 dropId = 12345 @0x70da1b
		// no crc on v48 (<83)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v48 pickup golden mismatch: got %v want %v", actual, expected)
	}
}

// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v83 ida=0xa09118
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v87 ida=0xa9e8f6
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v95 ida=0x9d5d50
// packet-audit:verify packet=drop/serverbound/DropPickUp version=jms_v185 ida=0xaedb0f
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v84 ida=0xa5342c
func TestPickUpRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := PickUp{fieldKey: 1, updateTime: 100, x: 50, y: 60, dropId: 12345, crc: 99}
			output := PickUp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.DropId() != input.DropId() {
				t.Errorf("dropId: got %v, want %v", output.DropId(), input.DropId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}
