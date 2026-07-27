package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte-fixture verify markers are added in Wave 2 once evidence is pinned.

// Per-version dispatcher mode bytes for the failure-arm family (task-183 Wave
// 1.1), taken from docs/tasks/task-183-cashshop-result-family/arm-catalog.md
// (MODERN-5 scope: gms_v83/v84/v87/v95, jms_v185 only — legacy modes are Wave 3).

var loadGiftFailedModes = map[string]byte{
	"GMS/v83": 0x4E, "GMS/v84": 0x51, "GMS/v87": 0x53, "GMS/v95": 0x5B, "JMS/v185": 0x51,
}

var loadWishFailedModes = map[string]byte{
	"GMS/v83": 0x50, "GMS/v84": 0x53, "GMS/v87": 0x55, "GMS/v95": 0x5D, "JMS/v185": 0x53,
}

var setWishFailedModes = map[string]byte{
	"GMS/v83": 0x56, "GMS/v84": 0x59, "GMS/v87": 0x5B, "GMS/v95": 0x63, "JMS/v185": 0x57,
}

var buyFailedModes = map[string]byte{
	"GMS/v83": 0x58, "GMS/v84": 0x5B, "GMS/v87": 0x5D, "GMS/v95": 0x65, "JMS/v185": 0x59,
}

var useCouponFailedModes = map[string]byte{
	"GMS/v83": 0x5C, "GMS/v84": 0x5F, "GMS/v87": 0x61, "GMS/v95": 0x69, "JMS/v185": 0x5D,
}

var giftFailedModes = map[string]byte{
	"GMS/v83": 0x5F, "GMS/v84": 0x62, "GMS/v87": 0x64, "GMS/v95": 0x6C, "JMS/v185": 0xA3,
}

var incTrunkCountFailedModes = map[string]byte{
	"GMS/v83": 0x63, "GMS/v84": 0x66, "GMS/v87": 0x68, "GMS/v95": 0x70, "JMS/v185": 0x64,
}

var incCharacterSlotCountFailedModes = map[string]byte{
	"GMS/v83": 0x65, "GMS/v84": 0x68, "GMS/v87": 0x6A, "GMS/v95": 0x72, "JMS/v185": 0x66,
}

// incBuyCharacterCountFailedModes: n-a in v83/v84/v87 (only present starting v95, per catalog).
var incBuyCharacterCountFailedModes = map[string]byte{
	"GMS/v95": 0x74, "JMS/v185": 0x68,
}

var enableEquipSlotExtFailedModes = map[string]byte{
	"GMS/v83": 0x67, "GMS/v84": 0x6A, "GMS/v87": 0x6C, "GMS/v95": 0x76, "JMS/v185": 0x6A,
}

var moveLToSFailedModes = map[string]byte{
	"GMS/v83": 0x69, "GMS/v84": 0x6C, "GMS/v87": 0x6E, "GMS/v95": 0x78, "JMS/v185": 0x6C,
}

var moveSToLFailedModes = map[string]byte{
	"GMS/v83": 0x6B, "GMS/v84": 0x6E, "GMS/v87": 0x70, "GMS/v95": 0x7A, "JMS/v185": 0x6E,
}

var destroyFailedModes = map[string]byte{
	"GMS/v83": 0x6D, "GMS/v84": 0x70, "GMS/v87": 0x72, "GMS/v95": 0x7C, "JMS/v185": 0x70,
}

var rebateFailedModes = map[string]byte{
	"GMS/v83": 0x86, "GMS/v84": 0x89, "GMS/v87": 0x8B, "GMS/v95": 0x97, "JMS/v185": 0x89,
}

var coupleFailedModes = map[string]byte{
	"GMS/v83": 0x88, "GMS/v84": 0x8B, "GMS/v87": 0x8D, "GMS/v95": 0x99, "JMS/v185": 0x8B,
}

var buyPackageFailedModes = map[string]byte{
	"GMS/v83": 0x8A, "GMS/v84": 0x8D, "GMS/v87": 0x8F, "GMS/v95": 0x9B, "JMS/v185": 0x8D,
}

var giftPackageFailedModes = map[string]byte{
	"GMS/v83": 0x8C, "GMS/v84": 0x8F, "GMS/v87": 0x91, "GMS/v95": 0x9D, "JMS/v185": 0x8F,
}

var buyNormalFailedModes = map[string]byte{
	"GMS/v83": 0x8E, "GMS/v84": 0x91, "GMS/v87": 0x93, "GMS/v95": 0x9F, "JMS/v185": 0x91,
}

var friendshipFailedModes = map[string]byte{
	"GMS/v83": 0x92, "GMS/v84": 0x95, "GMS/v87": 0x97, "GMS/v95": 0xA3, "JMS/v185": 0x95,
}

var purchaseRecordFailedModes = map[string]byte{
	"GMS/v83": 0x9B, "GMS/v84": 0x9E, "GMS/v87": 0xA4, "GMS/v95": 0xB0, "JMS/v185": 0x9E,
}

var transferWorldFailedModes = map[string]byte{
	"GMS/v83": 0xA1, "GMS/v84": 0xA4, "GMS/v87": 0xAA, "GMS/v95": 0xB6, "JMS/v185": 0xAF,
}

// gachaponOpenFailedModes / gachaponCopyFailedModes / changeMaplePointFailedModes:
// n-a in v83 and jms — present only v84/v87/v95 per catalog (§4 "v84 — 6 arms
// present that v83 lacks"; §5 jms-specific notes for CHANGE_MAPLE_POINT_FAILED).
var gachaponOpenFailedModes = map[string]byte{
	"GMS/v84": 0xA6, "GMS/v87": 0xAC, "GMS/v95": 0xB8,
}

var gachaponCopyFailedModes = map[string]byte{
	"GMS/v84": 0xA8, "GMS/v87": 0xAE, "GMS/v95": 0xBA,
}

var changeMaplePointFailedModes = map[string]byte{
	"GMS/v84": 0xAA, "GMS/v87": 0xB0, "GMS/v95": 0xBC,
}

// packet-audit:verify packet=cash/clientbound/CashBuyFailed version=gms_v83 ida=0x479985
// packet-audit:verify packet=cash/clientbound/CashBuyFailed version=gms_v84 ida=0x47cb23
// packet-audit:verify packet=cash/clientbound/CashBuyFailed version=gms_v87 ida=0x48515e
// packet-audit:verify packet=cash/clientbound/CashBuyFailed version=gms_v95 ida=0x4969f0
// packet-audit:verify packet=cash/clientbound/CashBuyFailed version=jms_v185 ida=0x48c441
func TestBuyFailedByteFixture(t *testing.T) {
	const errorCode = 0x03
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := buyFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no BUY_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuyFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("BUY_FAILED bytes: got %v, want %v", got, want)
			}
			output := BuyFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashLoadGiftFailed version=gms_v83 ida=0x4797c0
// packet-audit:verify packet=cash/clientbound/CashLoadGiftFailed version=gms_v84 ida=0x47c95e
// packet-audit:verify packet=cash/clientbound/CashLoadGiftFailed version=gms_v87 ida=0x484ee7
// packet-audit:verify packet=cash/clientbound/CashLoadGiftFailed version=gms_v95 ida=0x496960
// packet-audit:verify packet=cash/clientbound/CashLoadGiftFailed version=jms_v185 ida=0x48bfea
func TestLoadGiftFailedByteFixture(t *testing.T) {
	const errorCode = 0x01
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := loadGiftFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no LOAD_GIFT_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewLoadGiftFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("LOAD_GIFT_FAILED bytes: got %v, want %v", got, want)
			}
			output := LoadGiftFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashLoadWishFailed version=gms_v83 ida=0x479822
// packet-audit:verify packet=cash/clientbound/CashLoadWishFailed version=gms_v84 ida=0x47c9c0
// packet-audit:verify packet=cash/clientbound/CashLoadWishFailed version=gms_v87 ida=0x484f49
// packet-audit:verify packet=cash/clientbound/CashLoadWishFailed version=gms_v95 ida=0x496990
// packet-audit:verify packet=cash/clientbound/CashLoadWishFailed version=jms_v185 ida=0x48c04c
func TestLoadWishFailedByteFixture(t *testing.T) {
	const errorCode = 0x02
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := loadWishFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no LOAD_WISH_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewLoadWishFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("LOAD_WISH_FAILED bytes: got %v, want %v", got, want)
			}
			output := LoadWishFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashSetWishFailed version=gms_v83 ida=0x4798a7
// packet-audit:verify packet=cash/clientbound/CashSetWishFailed version=gms_v84 ida=0x47ca45
// packet-audit:verify packet=cash/clientbound/CashSetWishFailed version=gms_v87 ida=0x484fce
// packet-audit:verify packet=cash/clientbound/CashSetWishFailed version=gms_v95 ida=0x4969c0
// packet-audit:verify packet=cash/clientbound/CashSetWishFailed version=jms_v185 ida=0x48c0d1
func TestSetWishFailedByteFixture(t *testing.T) {
	const errorCode = 0x03
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := setWishFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no SET_WISH_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewSetWishFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("SET_WISH_FAILED bytes: got %v, want %v", got, want)
			}
			output := SetWishFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashUseCouponFailed version=gms_v83 ida=0x47a7db
// packet-audit:verify packet=cash/clientbound/CashUseCouponFailed version=gms_v84 ida=0x47d979
// packet-audit:verify packet=cash/clientbound/CashUseCouponFailed version=gms_v87 ida=0x485f93
// packet-audit:verify packet=cash/clientbound/CashUseCouponFailed version=gms_v95 ida=0x496f90
// packet-audit:verify packet=cash/clientbound/CashUseCouponFailed version=jms_v185 ida=0x48d390
func TestUseCouponFailedByteFixture(t *testing.T) {
	const errorCode = 0x04
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := useCouponFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no USE_COUPON_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewUseCouponFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("USE_COUPON_FAILED bytes: got %v, want %v", got, want)
			}
			output := UseCouponFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashGiftFailed version=gms_v83 ida=0x47a9a0
// packet-audit:verify packet=cash/clientbound/CashGiftFailed version=gms_v84 ida=0x47db3e
// packet-audit:verify packet=cash/clientbound/CashGiftFailed version=gms_v87 ida=0x486158
// packet-audit:verify packet=cash/clientbound/CashGiftFailed version=gms_v95 ida=0x497210
// packet-audit:verify packet=cash/clientbound/CashGiftFailed version=jms_v185 ida=0x48c2f3
func TestGiftFailedByteFixture(t *testing.T) {
	const errorCode = 0x05
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := giftFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no GIFT_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewGiftFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("GIFT_FAILED bytes: got %v, want %v", got, want)
			}
			output := GiftFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountFailed version=gms_v83 ida=0x47abca
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountFailed version=gms_v84 ida=0x47dd68
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountFailed version=gms_v87 ida=0x486393
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountFailed version=gms_v95 ida=0x4973d0
// packet-audit:verify packet=cash/clientbound/CashIncTrunkCountFailed version=jms_v185 ida=0x48d71e
func TestIncTrunkCountFailedByteFixture(t *testing.T) {
	const errorCode = 0x06
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := incTrunkCountFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no INC_TRUNK_COUNT_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewIncTrunkCountFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("INC_TRUNK_COUNT_FAILED bytes: got %v, want %v", got, want)
			}
			output := IncTrunkCountFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountFailed version=gms_v83 ida=0x47ac9b
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountFailed version=gms_v84 ida=0x47de39
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountFailed version=gms_v87 ida=0x48646d
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountFailed version=gms_v95 ida=0x497410
// packet-audit:verify packet=cash/clientbound/CashIncCharacterSlotCountFailed version=jms_v185 ida=0x48d7ef
func TestIncCharacterSlotCountFailedByteFixture(t *testing.T) {
	const errorCode = 0x07
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := incCharacterSlotCountFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no INC_CHARACTER_SLOT_COUNT_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewIncCharacterSlotCountFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("INC_CHARACTER_SLOT_COUNT_FAILED bytes: got %v, want %v", got, want)
			}
			output := IncCharacterSlotCountFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashIncBuyCharacterCountFailed version=gms_v95 ida=0x497450
// packet-audit:verify packet=cash/clientbound/CashIncBuyCharacterCountFailed version=jms_v185 ida=0x48d871
func TestIncBuyCharacterCountFailedByteFixture(t *testing.T) {
	const errorCode = 0x08
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := incBuyCharacterCountFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no INC_BUY_CHARACTER_COUNT_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewIncBuyCharacterCountFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("INC_BUY_CHARACTER_COUNT_FAILED bytes: got %v, want %v", got, want)
			}
			output := IncBuyCharacterCountFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtFailed version=gms_v83 ida=0x47aea2
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtFailed version=gms_v84 ida=0x47e040
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtFailed version=gms_v87 ida=0x486674
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtFailed version=gms_v95 ida=0x4976f0
// packet-audit:verify packet=cash/clientbound/CashEnableEquipSlotExtFailed version=jms_v185 ida=0x48da78
func TestEnableEquipSlotExtFailedByteFixture(t *testing.T) {
	const errorCode = 0x09
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := enableEquipSlotExtFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no ENABLE_EQUIP_SLOT_EXT_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEnableEquipSlotExtFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("ENABLE_EQUIP_SLOT_EXT_FAILED bytes: got %v, want %v", got, want)
			}
			output := EnableEquipSlotExtFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashMoveLToSFailed version=gms_v83 ida=0x47b18c
// packet-audit:verify packet=cash/clientbound/CashMoveLToSFailed version=gms_v84 ida=0x47e32a
// packet-audit:verify packet=cash/clientbound/CashMoveLToSFailed version=gms_v87 ida=0x486962
// packet-audit:verify packet=cash/clientbound/CashMoveLToSFailed version=gms_v95 ida=0x497730
// packet-audit:verify packet=cash/clientbound/CashMoveLToSFailed version=jms_v185 ida=0x48dd65
func TestMoveLToSFailedByteFixture(t *testing.T) {
	const errorCode = 0x0A
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := moveLToSFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no MOVE_L_TO_S_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMoveLToSFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("MOVE_L_TO_S_FAILED bytes: got %v, want %v", got, want)
			}
			output := MoveLToSFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashMoveSToLFailed version=gms_v83 ida=0x47b401
// packet-audit:verify packet=cash/clientbound/CashMoveSToLFailed version=gms_v84 ida=0x47e59f
// packet-audit:verify packet=cash/clientbound/CashMoveSToLFailed version=gms_v87 ida=0x486bdf
// packet-audit:verify packet=cash/clientbound/CashMoveSToLFailed version=gms_v95 ida=0x497920
// packet-audit:verify packet=cash/clientbound/CashMoveSToLFailed version=jms_v185 ida=0x48dfdb
func TestMoveSToLFailedByteFixture(t *testing.T) {
	const errorCode = 0x0B
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := moveSToLFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no MOVE_S_TO_L_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMoveSToLFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("MOVE_S_TO_L_FAILED bytes: got %v, want %v", got, want)
			}
			output := MoveSToLFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashDestroyFailed version=gms_v83 ida=0x47b4c5
// packet-audit:verify packet=cash/clientbound/CashDestroyFailed version=gms_v84 ida=0x47e663
// packet-audit:verify packet=cash/clientbound/CashDestroyFailed version=gms_v87 ida=0x486ca3
// packet-audit:verify packet=cash/clientbound/CashDestroyFailed version=gms_v95 ida=0x497950
// packet-audit:verify packet=cash/clientbound/CashDestroyFailed version=jms_v185 ida=0x48e09f
func TestDestroyFailedByteFixture(t *testing.T) {
	const errorCode = 0x0C
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := destroyFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no DESTROY_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewDestroyFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("DESTROY_FAILED bytes: got %v, want %v", got, want)
			}
			output := DestroyFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashRebateFailed version=gms_v83 ida=0x47b5e4
// packet-audit:verify packet=cash/clientbound/CashRebateFailed version=gms_v84 ida=0x47e782
// packet-audit:verify packet=cash/clientbound/CashRebateFailed version=gms_v87 ida=0x486dc2
// packet-audit:verify packet=cash/clientbound/CashRebateFailed version=gms_v95 ida=0x497ad0
// packet-audit:verify packet=cash/clientbound/CashRebateFailed version=jms_v185 ida=0x48e1be
func TestRebateFailedByteFixture(t *testing.T) {
	const errorCode = 0x0D
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := rebateFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no REBATE_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewRebateFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("REBATE_FAILED bytes: got %v, want %v", got, want)
			}
			output := RebateFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashCoupleFailed version=gms_v83 ida=0x47b8c2
// packet-audit:verify packet=cash/clientbound/CashCoupleFailed version=gms_v84 ida=0x47ea60
// packet-audit:verify packet=cash/clientbound/CashCoupleFailed version=gms_v87 ida=0x4870a3
// packet-audit:verify packet=cash/clientbound/CashCoupleFailed version=gms_v95 ida=0x497d20
// packet-audit:verify packet=cash/clientbound/CashCoupleFailed version=jms_v185 ida=0x48e4a0
func TestCoupleFailedByteFixture(t *testing.T) {
	const errorCode = 0x0E
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := coupleFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no COUPLE_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCoupleFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("COUPLE_FAILED bytes: got %v, want %v", got, want)
			}
			output := CoupleFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashBuyPackageFailed version=gms_v83 ida=0x479ba0
// packet-audit:verify packet=cash/clientbound/CashBuyPackageFailed version=gms_v84 ida=0x47cd3e
// packet-audit:verify packet=cash/clientbound/CashBuyPackageFailed version=gms_v87 ida=0x485379
// packet-audit:verify packet=cash/clientbound/CashBuyPackageFailed version=gms_v95 ida=0x496d40
// packet-audit:verify packet=cash/clientbound/CashBuyPackageFailed version=jms_v185 ida=0x48c718
func TestBuyPackageFailedByteFixture(t *testing.T) {
	const errorCode = 0x0F
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := buyPackageFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no BUY_PACKAGE_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuyPackageFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("BUY_PACKAGE_FAILED bytes: got %v, want %v", got, want)
			}
			output := BuyPackageFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashGiftPackageFailed version=gms_v83 ida=0x479d10
// packet-audit:verify packet=cash/clientbound/CashGiftPackageFailed version=gms_v84 ida=0x47ceae
// packet-audit:verify packet=cash/clientbound/CashGiftPackageFailed version=gms_v87 ida=0x4854e9
// packet-audit:verify packet=cash/clientbound/CashGiftPackageFailed version=gms_v95 ida=0x496f20
// packet-audit:verify packet=cash/clientbound/CashGiftPackageFailed version=jms_v185 ida=0x48c8ec
func TestGiftPackageFailedByteFixture(t *testing.T) {
	const errorCode = 0x10
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := giftPackageFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no GIFT_PACKAGE_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewGiftPackageFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("GIFT_PACKAGE_FAILED bytes: got %v, want %v", got, want)
			}
			output := GiftPackageFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashBuyNormalFailed version=gms_v83 ida=0x47b71c
// packet-audit:verify packet=cash/clientbound/CashBuyNormalFailed version=gms_v84 ida=0x47e8ba
// packet-audit:verify packet=cash/clientbound/CashBuyNormalFailed version=gms_v87 ida=0x486efd
// packet-audit:verify packet=cash/clientbound/CashBuyNormalFailed version=gms_v95 ida=0x497b00
// packet-audit:verify packet=cash/clientbound/CashBuyNormalFailed version=jms_v185 ida=0x48e2f9
func TestBuyNormalFailedByteFixture(t *testing.T) {
	const errorCode = 0x11
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := buyNormalFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no BUY_NORMAL_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuyNormalFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("BUY_NORMAL_FAILED bytes: got %v, want %v", got, want)
			}
			output := BuyNormalFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashFriendshipFailed version=gms_v83 ida=0x47ba70
// packet-audit:verify packet=cash/clientbound/CashFriendshipFailed version=gms_v84 ida=0x47ec0e
// packet-audit:verify packet=cash/clientbound/CashFriendshipFailed version=gms_v87 ida=0x487251
// packet-audit:verify packet=cash/clientbound/CashFriendshipFailed version=gms_v95 ida=0x497f40
// packet-audit:verify packet=cash/clientbound/CashFriendshipFailed version=jms_v185 ida=0x48e64f
func TestFriendshipFailedByteFixture(t *testing.T) {
	const errorCode = 0x12
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := friendshipFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no FRIENDSHIP_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewFriendshipFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("FRIENDSHIP_FAILED bytes: got %v, want %v", got, want)
			}
			output := FriendshipFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordFailed version=gms_v83 ida=0x47c0fc
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordFailed version=gms_v84 ida=0x47f29a
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordFailed version=gms_v87 ida=0x4878dd
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordFailed version=gms_v95 ida=0x494070
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordFailed version=jms_v185 ida=0x48e79a
func TestPurchaseRecordFailedByteFixture(t *testing.T) {
	const errorCode = 0x13
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := purchaseRecordFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no PURCHASE_RECORD_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewPurchaseRecordFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("PURCHASE_RECORD_FAILED bytes: got %v, want %v", got, want)
			}
			output := PurchaseRecordFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashTransferWorldFailed version=gms_v83 ida=0x47c072
// packet-audit:verify packet=cash/clientbound/CashTransferWorldFailed version=gms_v84 ida=0x47f210
// packet-audit:verify packet=cash/clientbound/CashTransferWorldFailed version=gms_v87 ida=0x487853
// packet-audit:verify packet=cash/clientbound/CashTransferWorldFailed version=gms_v95 ida=0x498370
// packet-audit:verify packet=cash/clientbound/CashTransferWorldFailed version=jms_v185 ida=0x48ea61
func TestTransferWorldFailedByteFixture(t *testing.T) {
	const errorCode = 0x14
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := transferWorldFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no TRANSFER_WORLD_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewTransferWorldFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("TRANSFER_WORLD_FAILED bytes: got %v, want %v", got, want)
			}
			output := TransferWorldFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashGachaponOpenFailed version=gms_v84 ida=0x47faa2
// packet-audit:verify packet=cash/clientbound/CashGachaponOpenFailed version=gms_v87 ida=0x488100
// packet-audit:verify packet=cash/clientbound/CashGachaponOpenFailed version=gms_v95 ida=0x4962b0
func TestGachaponOpenFailedByteFixture(t *testing.T) {
	const errorCode = 0x15
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := gachaponOpenFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no GACHAPON_OPEN_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewGachaponOpenFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("GACHAPON_OPEN_FAILED bytes: got %v, want %v", got, want)
			}
			output := GachaponOpenFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// packet-audit:verify packet=cash/clientbound/CashGachaponCopyFailed version=gms_v84 ida=0x47fb6f
// packet-audit:verify packet=cash/clientbound/CashGachaponCopyFailed version=gms_v87 ida=0x4881cd
// packet-audit:verify packet=cash/clientbound/CashGachaponCopyFailed version=gms_v95 ida=0x4962f0
func TestGachaponCopyFailedByteFixture(t *testing.T) {
	const errorCode = 0x16
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := gachaponCopyFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no GACHAPON_COPY_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewGachaponCopyFailed(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("GACHAPON_COPY_FAILED bytes: got %v, want %v", got, want)
			}
			output := GachaponCopyFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ErrorCode() != errorCode {
				t.Errorf("round-trip mismatch: mode %v err %v", output.Mode(), output.ErrorCode())
			}
		})
	}
}

// TestChangeMaplePointFailedByteFixture exercises the bodyless shape: the wire
// carries ONLY the mode byte (RE-confirmed zero further Decode calls in the
// handler — arm-catalog.md CHANGE_MAPLE_POINT_FAILED row).
// packet-audit:verify packet=cash/clientbound/CashChangeMaplePointFailed version=gms_v84 ida=0x47fda5
// packet-audit:verify packet=cash/clientbound/CashChangeMaplePointFailed version=gms_v87 ida=0x488406
// packet-audit:verify packet=cash/clientbound/CashChangeMaplePointFailed version=gms_v95 ida=0x495910
func TestChangeMaplePointFailedByteFixture(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := changeMaplePointFailedModes[variantKey(v)]
			if !ok {
				t.Skipf("no CHANGE_MAPLE_POINT_FAILED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewChangeMaplePointFailed(mode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode}
			if !bytesEqual(got, want) {
				t.Errorf("CHANGE_MAPLE_POINT_FAILED bytes: got %v, want %v", got, want)
			}
			output := ChangeMaplePointFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode {
				t.Errorf("mode: got %v, want %v", output.Mode(), mode)
			}
		})
	}
}
