package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestDropDestroyByteOutputV79 pins the gms_v79 REMOVE_ITEM_FROM_MAP (op 0x0F7)
// clientbound wire for the byte-exact arms. IDA-verified client decode
// (GMS_v79_1_DEVM.exe, port 13340) — CDropPool::OnDropLeaveField @0x4f328f:
//
//	v2 = CInPacket::Decode1(a2)     @0x4f32af → destroyType byte.
//	v83 = CInPacket::Decode4(a2)    @0x4f32bb → dropId uint32-LE.
//	if (v2==2||v2==3||v2==5) Decode4 @0x4f3326 → pickupCharId uint32-LE.
//	else if (v2==4)         Decode2 @0x4f32f7 → explode tLeaveDelay int16-LE.
//	(types 0/1: no trailing field.)
//
// NOTE: the type-5 (pet pickup) arm reads ONE EXTRA byte (CInPacket::Decode1
// @0x4f33b2) in v79 — and v83 (@0x506590) reads the same single Decode1 — but
// v95 (@0x511e20) widened that trailing extra to a Decode4 (int). The codec
// emits the v95 int4 shape (petPickupExtra) unconditionally; gating it for
// legacy versions needs the v87 boundary (v87 IDB not loaded), so the type-5
// arm is intentionally excluded from the v79 golden below. The byte-exact
// arms (0/2/4) match the v79 client exactly.
//
// packet-audit:verify packet=drop/clientbound/DropDestroy version=gms_v79 ida=0x4f328f
func TestDropDestroyByteOutputV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)

	// type 0 (expire): byte(0) + int(dropId=9001=0x2329).
	exp0 := []byte{0x00, 0x29, 0x23, 0x00, 0x00}
	if got := NewDropDestroy(9001, DropDestroyTypeExpire, 0, -1).Encode(l, ctx)(nil); !bytes.Equal(got, exp0) {
		t.Errorf("v79 destroy expire golden mismatch: got %v want %v", got, exp0)
	}

	// type 2 (pickup): byte(2) + int(dropId) + int(charId=1234=0x4D2).
	exp2 := []byte{0x02, 0x29, 0x23, 0x00, 0x00, 0xD2, 0x04, 0x00, 0x00}
	if got := NewDropDestroy(9001, DropDestroyTypePickUp, 1234, -1).Encode(l, ctx)(nil); !bytes.Equal(got, exp2) {
		t.Errorf("v79 destroy pickup golden mismatch: got %v want %v", got, exp2)
	}

	// type 4 (explode): byte(4) + int(dropId) + int16(delay=500=0x1F4).
	exp4 := []byte{0x04, 0x29, 0x23, 0x00, 0x00, 0xF4, 0x01}
	if got := NewDropDestroyExplode(9001, 500).Encode(l, ctx)(nil); !bytes.Equal(got, exp4) {
		t.Errorf("v79 destroy explode golden mismatch: got %v want %v", got, exp4)
	}
}

// TestDropDestroyByteOutputV72 pins the gms_v72 REMOVE_ITEM_FROM_MAP clientbound
// wire for the byte-exact arms. IDA-verified client decode
// (GMS_v72.1_U_DEVM.exe, port 13339) — CDropPool::OnDropLeaveField @0x4ea5fc:
//
//	v2  = CInPacket::Decode1(a2)     @0x4ea61c → destroyType byte.
//	v83 = CInPacket::Decode4(a2)     @0x4ea628 → dropId uint32-LE.
//	if (v2==2||v2==3||v2==5) Decode4 @0x4ea693 → pickupCharId uint32-LE.
//	else if (v2==4)         Decode2  @0x4ea664 → explode tLeaveDelay int16-LE.
//	(types 0/1: no trailing field.)
//
// Byte-identical to the verified v79 wire for the 0/2/4 arms. As with v79, the
// type-5 (pet pickup) arm is excluded from this golden: the codec emits the v95
// int4 petPickupExtra shape unconditionally and gating it needs the v87 boundary
// (out of scope here). The byte-exact arms (0/2/4) match the v72 client exactly.
//
// packet-audit:verify packet=drop/clientbound/DropDestroy version=gms_v72 ida=0x4ea5fc
func TestDropDestroyByteOutputV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)

	// type 0 (expire): byte(0) + int(dropId=9001=0x2329).
	exp0 := []byte{0x00, 0x29, 0x23, 0x00, 0x00}
	if got := NewDropDestroy(9001, DropDestroyTypeExpire, 0, -1).Encode(l, ctx)(nil); !bytes.Equal(got, exp0) {
		t.Errorf("v72 destroy expire golden mismatch: got %v want %v", got, exp0)
	}

	// type 2 (pickup): byte(2) + int(dropId) + int(charId=1234=0x4D2).
	exp2 := []byte{0x02, 0x29, 0x23, 0x00, 0x00, 0xD2, 0x04, 0x00, 0x00}
	if got := NewDropDestroy(9001, DropDestroyTypePickUp, 1234, -1).Encode(l, ctx)(nil); !bytes.Equal(got, exp2) {
		t.Errorf("v72 destroy pickup golden mismatch: got %v want %v", got, exp2)
	}

	// type 4 (explode): byte(4) + int(dropId) + int16(delay=500=0x1F4).
	exp4 := []byte{0x04, 0x29, 0x23, 0x00, 0x00, 0xF4, 0x01}
	if got := NewDropDestroyExplode(9001, 500).Encode(l, ctx)(nil); !bytes.Equal(got, exp4) {
		t.Errorf("v72 destroy explode golden mismatch: got %v want %v", got, exp4)
	}
}

// packet-audit:verify packet=drop/clientbound/DropDestroy version=gms_v83 ida=0x506590
// packet-audit:verify packet=drop/clientbound/DropDestroy version=gms_v87 ida=0x5287e3
// packet-audit:verify packet=drop/clientbound/DropDestroy version=gms_v95 ida=0x511e20
// packet-audit:verify packet=drop/clientbound/DropDestroy version=jms_v185 ida=0x537726
// packet-audit:verify packet=drop/clientbound/DropDestroy version=gms_v84 ida=0x50f409
func TestDropDestroyPickUp(t *testing.T) {
	input := NewDropDestroy(9001, DropDestroyTypePickUp, 1234, 2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestDropDestroyExpire(t *testing.T) {
	input := NewDropDestroy(9001, DropDestroyTypeExpire, 0, -1)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestDropDestroyExplode pins the v95 wire shape for destroyType == 4:
// byte(4) + int(dropId) + int16(tLeaveDelay) = 7 bytes. The legacy
// NewDropDestroy(dropId, 4, charId, -1) path emits the same shape with
// delay = 0 since callers historically passed characterId=0/petSlot=-1.
func TestDropDestroyExplode(t *testing.T) {
	input := NewDropDestroyExplode(9001, 500)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 1)
	bytes := input.Encode(l, ctx)(nil)
	if len(bytes) != 7 {
		t.Errorf("explode encode: got %d bytes, want 7 (byte type + uint32 dropId + int16 delay)", len(bytes))
	}
}

// TestDropDestroyPetPickUp pins the v95 wire shape for destroyType == 5:
// byte(5) + int(dropId) + int(pickupCharId) + int(petPickupExtra) = 13 bytes.
// Legacy NewDropDestroy with petSlot >= 0 widens the petSlot to the
// int4 v95 reads inside the case 5 body.
func TestDropDestroyPetPickUp(t *testing.T) {
	input := NewDropDestroy(9001, DropDestroyTypePetPickUp, 1234, 2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 1)
	bytes := input.Encode(l, ctx)(nil)
	if len(bytes) != 13 {
		t.Errorf("pet pickup encode: got %d bytes, want 13 (byte type + uint32 dropId + uint32 charId + uint32 extra)", len(bytes))
	}
}
