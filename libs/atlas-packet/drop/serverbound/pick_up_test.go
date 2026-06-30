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
