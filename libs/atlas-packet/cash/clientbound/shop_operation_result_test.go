package clientbound

import (
	"fmt"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/clientbound/CashInventoryCapacitySuccess version=gms_v95 ida=0x497270
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacitySuccess version=gms_v83 ida=0x47a9fa
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacitySuccess version=gms_v84 ida=0x47db98
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacitySuccess version=gms_v87 ida=0x4861b2
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacitySuccess version=jms_v185 ida=0x48d54e
// packet-audit:verify packet=cash/clientbound/CashLoadInventoryFailure version=gms_v95 ida=0x4969f0
// packet-audit:verify packet=cash/clientbound/CashLoadInventoryFailure version=gms_v83 ida=0x47957c
// packet-audit:verify packet=cash/clientbound/CashLoadInventoryFailure version=gms_v84 ida=0x47c71a
// packet-audit:verify packet=cash/clientbound/CashLoadInventoryFailure version=gms_v87 ida=0x484ca3
// packet-audit:verify packet=cash/clientbound/CashLoadInventoryFailure version=jms_v185 ida=0x48bda6
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacityFailed version=gms_v95 ida=0x497390
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacityFailed version=gms_v83 ida=0x47aaee
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacityFailed version=gms_v84 ida=0x47dc8c
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacityFailed version=gms_v87 ida=0x4862ae
// packet-audit:verify packet=cash/clientbound/CashInventoryCapacityFailed version=jms_v185 ida=0x48d642
// packet-audit:verify packet=cash/clientbound/CashWishListLoad version=gms_v95 ida=0x494020
// packet-audit:verify packet=cash/clientbound/CashWishListLoad version=gms_v83 ida=0x4797e2
// packet-audit:verify packet=cash/clientbound/CashWishListLoad version=gms_v84 ida=0x47c980
// packet-audit:verify packet=cash/clientbound/CashWishListLoad version=gms_v87 ida=0x484f09
// packet-audit:verify packet=cash/clientbound/CashWishListLoad version=jms_v185 ida=0x48c00c
// packet-audit:verify packet=cash/clientbound/CashWishListUpdate version=gms_v95 ida=0x494d60
// packet-audit:verify packet=cash/clientbound/CashWishListUpdate version=gms_v83 ida=0x479844
// packet-audit:verify packet=cash/clientbound/CashWishListUpdate version=gms_v84 ida=0x47c9e2
// packet-audit:verify packet=cash/clientbound/CashWishListUpdate version=gms_v87 ida=0x484f6b
// packet-audit:verify packet=cash/clientbound/CashWishListUpdate version=jms_v185 ida=0x48c06e
// loadInventoryFailureModes are the per-version dispatcher mode bytes for the
// LOAD_INVENTORY_FAILURE case of CCashShop::OnCashItemResult (handler
// OnCashItemResLoadLockerFailed), taken from
// docs/packets/dispatchers/cash_shop_operation.yaml (IDA-verified): gms_v83 76
// (0x4C), gms_v84 79, gms_v87 81, gms_v95 89, jms_v185 79.
var loadInventoryFailureModes = map[string]byte{
	"GMS/v83": 76, "GMS/v84": 79, "GMS/v87": 81, "GMS/v95": 89, "JMS/v185": 79,
}

func TestLoadInventoryFailureByteFixture(t *testing.T) {
	const errorCode = 0x01
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := loadInventoryFailureModes[variantKey(v)]
			if !ok {
				t.Skipf("no LOAD_INVENTORY_FAILURE mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewLoadInventoryFailure(mode, errorCode)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, errorCode}
			if !bytesEqual(got, want) {
				t.Errorf("LOAD_INVENTORY_FAILURE bytes: got %v, want %v", got, want)
			}
			output := LoadInventoryFailure{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode {
				t.Errorf("mode: got %v, want %v", output.Mode(), mode)
			}
			if output.ErrorCode() != errorCode {
				t.Errorf("errorCode: got %v, want %v", output.ErrorCode(), errorCode)
			}
		})
	}
}

func TestInventoryCapacitySuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewInventoryCapacitySuccess(0x30, 1, 96)
			output := InventoryCapacitySuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Capacity() != input.Capacity() {
				t.Errorf("capacity: got %v, want %v", output.Capacity(), input.Capacity())
			}
		})
	}
}

func TestInventoryCapacityFailedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewInventoryCapacityFailed(0x31, 0x02)
			output := InventoryCapacityFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.ErrorCode() != input.ErrorCode() {
				t.Errorf("errorCode: got %v, want %v", output.ErrorCode(), input.ErrorCode())
			}
		})
	}
}

// wishListBody computes the expected wire bytes for a wishlist arm: the leading
// mode byte followed by 10 little-endian int32 SNs (the first len(sns) populated,
// the rest zero-padded), matching CInPacket::DecodeBuffer(this+wishbuf, 40) in
// CCashShop::OnCashItemResLoad/SetWishDone.
func wishListBody(mode byte, sns []uint32) []byte {
	out := []byte{mode}
	for i := 0; i < 10; i++ {
		var v uint32
		if i < len(sns) {
			v = sns[i]
		}
		out = append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	}
	return out
}

// wishLoadModes / wishUpdateModes are the per-version dispatcher mode bytes for
// the LOAD_WISH_DONE / SET_WISH_DONE cases of CCashShop::OnCashItemResult, taken
// from docs/packets/dispatchers/cash_shop_operation.yaml (IDA-verified).
var wishLoadModes = map[string]byte{
	"GMS/v83": 0x4F, "GMS/v84": 0x52, "GMS/v87": 0x54, "GMS/v95": 0x5C, "JMS/v185": 0x52,
}

var wishUpdateModes = map[string]byte{
	"GMS/v83": 0x55, "GMS/v84": 0x58, "GMS/v87": 0x5A, "GMS/v95": 0x62, "JMS/v185": 0x56,
}

func variantKey(v pt.TenantVariant) string {
	return fmt.Sprintf("%s/v%d", v.Region, v.MajorVersion)
}

func TestWishListLoadByteFixture(t *testing.T) {
	sns := []uint32{101, 102, 103, 104, 105}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := wishLoadModes[variantKey(v)]
			if !ok {
				t.Skipf("no LOAD_WISH mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewWishListLoad(mode, sns)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := wishListBody(mode, sns)
			if !bytesEqual(got, want) {
				t.Errorf("LOAD_WISHLIST bytes: got %v, want %v", got, want)
			}
			// round-trip the discrete struct
			output := WishListLoad{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode {
				t.Errorf("mode: got %v, want %v", output.Mode(), mode)
			}
		})
	}
}

func TestWishListUpdateByteFixture(t *testing.T) {
	sns := []uint32{201, 202, 203}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := wishUpdateModes[variantKey(v)]
			if !ok {
				t.Skipf("no SET_WISH mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewWishListUpdate(mode, sns)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := wishListBody(mode, sns)
			if !bytesEqual(got, want) {
				t.Errorf("UPDATE_WISHLIST bytes: got %v, want %v", got, want)
			}
			output := WishListUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode {
				t.Errorf("mode: got %v, want %v", output.Mode(), mode)
			}
		})
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
