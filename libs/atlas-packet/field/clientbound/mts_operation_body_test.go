package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Per-mode body arms of CITC::OnNormalItemResult (MTS_OPERATION), task-096
// iteration 1. The body shapes are version-stable across gms_v83/v84/v87/v95
// (IDA-confirmed identical read order; only the dispatcher mode bytes and
// sub-handler addresses shift). jms_v185 has NO CITC op (VERSION-ABSENT) so it
// is not marked. The per-version ida= addresses below pin each arm's
// sub-handler in the matching dispatcher.

// MtsResultRegisterSaleEntryFailed (0x1E) — Decode1(reason); ONLY when
// reason==0x48 a trailing Decode2 sale-limit short. Decompiled identical in all
// four versions (v83 sub_5A4581 / v84 sub_5B4A38 / v87 0x5d4640 / v95 0x576b80).
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryFailed version=gms_v83 ida=0x5a4581
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryFailed version=gms_v84 ida=0x5b4a38
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryFailed version=gms_v87 ida=0x5d4640
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryFailed version=gms_v95 ida=0x576b80
func TestMtsResultRegisterSaleEntryFailedGolden(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 0)

	// reason != 0x48: only mode + reason on the wire (no trailing short).
	t.Run("PlainReason", func(t *testing.T) {
		input := NewMtsResultRegisterSaleEntryFailed(0x42, 0) // 'B' -> NoticeFailReason only
		expected := []byte{0x1E, 0x42}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})

	// reason == 0x48: mode + reason + Decode2 sale-limit short (little-endian).
	t.Run("SaleLimitReason", func(t *testing.T) {
		input := NewMtsResultRegisterSaleEntryFailed(0x48, 0x0064) // 'H', limit 100
		expected := []byte{0x1E, 0x48, 0x64, 0x00}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})
}

func TestMtsResultRegisterSaleEntryFailedRoundTrip(t *testing.T) {
	for _, in := range []MtsResultRegisterSaleEntryFailed{
		NewMtsResultRegisterSaleEntryFailed(0x42, 0),
		NewMtsResultRegisterSaleEntryFailed(0x48, 0x1234),
	} {
		input := in
		for _, v := range test.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				output := MtsResultRegisterSaleEntryFailed{}
				test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Mode() != input.Mode() {
					t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
				}
				if output.Reason() != input.Reason() {
					t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
				}
				if output.SaleLimit() != input.SaleLimit() {
					t.Errorf("saleLimit: got %v, want %v", output.SaleLimit(), input.SaleLimit())
				}
			})
		}
	}
}

// MtsResultSuccessBidInfo (0x3E) — Decode1(soldFlag) + Decode4(itemId); ONLY when
// itemId>0 a trailing Decode4(price) + DecodeBuffer(8) FILETIME contract date.
// Decompiled identical in all four versions (v83 sub_5A52DE / v84 sub_5B5795 /
// v87 0x5d5398 / v95 0x577000).
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultSuccessBidInfo version=gms_v83 ida=0x5a52de
// packet-audit:verify packet=field/clientbound/FieldMtsResultSuccessBidInfo version=gms_v84 ida=0x5b5795
// packet-audit:verify packet=field/clientbound/FieldMtsResultSuccessBidInfo version=gms_v87 ida=0x5d5398
// packet-audit:verify packet=field/clientbound/FieldMtsResultSuccessBidInfo version=gms_v95 ida=0x577000
func TestMtsResultSuccessBidInfoGolden(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 0)

	// itemId <= 0: body ends after mode + soldFlag + itemId (no notice path).
	t.Run("NoItem", func(t *testing.T) {
		input := NewMtsResultSuccessBidInfo(1, 0, 0, [8]byte{})
		expected := []byte{0x3E, 0x01, 0x00, 0x00, 0x00, 0x00}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})

	// itemId > 0: mode + soldFlag + itemId + price + 8-byte FILETIME buffer.
	t.Run("WithItem", func(t *testing.T) {
		date := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
		input := NewMtsResultSuccessBidInfo(1, 0x00204FCE, 0x000F4240, date) // itemId 2117070, price 1,000,000
		expected := []byte{
			0x3E,                   // mode
			0x01,                   // soldFlag
			0xCE, 0x4F, 0x20, 0x00, // itemId 0x00204FCE little-endian
			0x40, 0x42, 0x0F, 0x00, // price 0x000F4240 little-endian
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, // contract date buffer
		}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})
}

func TestMtsResultSuccessBidInfoRoundTrip(t *testing.T) {
	date := [8]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x11, 0x22, 0x33, 0x44}
	for _, in := range []MtsResultSuccessBidInfo{
		NewMtsResultSuccessBidInfo(0, 0, 0, [8]byte{}),
		NewMtsResultSuccessBidInfo(1, 0x00204FCE, 0x000F4240, date),
	} {
		input := in
		for _, v := range test.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				output := MtsResultSuccessBidInfo{}
				test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Mode() != input.Mode() {
					t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
				}
				if output.SoldFlag() != input.SoldFlag() {
					t.Errorf("soldFlag: got %v, want %v", output.SoldFlag(), input.SoldFlag())
				}
				if output.ItemId() != input.ItemId() {
					t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
				}
				if output.Price() != input.Price() {
					t.Errorf("price: got %v, want %v", output.Price(), input.Price())
				}
				if output.ContractDate() != input.ContractDate() {
					t.Errorf("contractDate: got %v, want %v", output.ContractDate(), input.ContractDate())
				}
			})
		}
	}
}
