package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Discrete per-mode golden + round-trip tests for the notice-only ("Empty-shape")
// arms of CITC::OnNormalItemResult (MTS_OPERATION). Each arm fixes its own mode
// byte and writes exactly that byte (the sub-handler reads NOTHING after the
// dispatcher Decode1(mode) — StringPool::GetString + CUtilDlg::Notice only). The
// mode bytes are version-stable; per-version sub-handler addresses are cited in
// each verify marker (dispatcher: v83 0x5a4311 / v84 0x5b47c8 / v87 0x5d43d0 /
// v95 0x5771d0). jms_v185 has NO CITC op (VERSION-ABSENT, unmarked).

// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryDone version=gms_v83 ida=0x5a4674
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryDone version=gms_v84 ida=0x5b4b64
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryDone version=gms_v87 ida=0x5d4748
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryDone version=gms_v95 ida=0x575cd0
func TestMtsResultRegisterSaleEntryDone(t *testing.T) {
	input := NewMtsResultRegisterSaleEntryDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x1D} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultRegisterSaleEntryDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishDone version=gms_v83 ida=0x5a46b2
// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishDone version=gms_v84 ida=0x5b4ba2
// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishDone version=gms_v87 ida=0x5d4786
// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishDone version=gms_v95 ida=0x575d20
func TestMtsResultSaleCurrentItemToWishDone(t *testing.T) {
	input := NewMtsResultSaleCurrentItemToWishDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x1F} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultSaleCurrentItemToWishDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemDone version=gms_v83 ida=0x5a4d14
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemDone version=gms_v84 ida=0x5b5204
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemDone version=gms_v87 ida=0x5d4e04
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemDone version=gms_v95 ida=0x576030
func TestMtsResultCancelSaleItemDone(t *testing.T) {
	input := NewMtsResultCancelSaleItemDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x25} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultCancelSaleItemDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimDone version=gms_v83 ida=0x5a4dfc
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimDone version=gms_v84 ida=0x5b52ec
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimDone version=gms_v87 ida=0x5d4eef
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimDone version=gms_v95 ida=0x576140
func TestMtsResultSetZzimDone(t *testing.T) {
	input := NewMtsResultSetZzimDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x29} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultSetZzimDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimFailed version=gms_v83 ida=0x5a4e31
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimFailed version=gms_v84 ida=0x5b5321
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimFailed version=gms_v87 ida=0x5d4f24
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimFailed version=gms_v95 ida=0x576180
func TestMtsResultSetZzimFailed(t *testing.T) {
	input := NewMtsResultSetZzimFailed()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2A} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultSetZzimFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimDone version=gms_v83 ida=0x5a4e66
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimDone version=gms_v84 ida=0x5b5356
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimDone version=gms_v87 ida=0x5d4f59
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimDone version=gms_v95 ida=0x5761c0
func TestMtsResultDeleteZzimDone(t *testing.T) {
	input := NewMtsResultDeleteZzimDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2B} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultDeleteZzimDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimFailed version=gms_v83 ida=0x5a4e91
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimFailed version=gms_v84 ida=0x5b5381
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimFailed version=gms_v87 ida=0x5d4f84
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimFailed version=gms_v95 ida=0x5761f0
func TestMtsResultDeleteZzimFailed(t *testing.T) {
	input := NewMtsResultDeleteZzimFailed()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2C} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultDeleteZzimFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListFailed version=gms_v83 ida=0x5a4fdc
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListFailed version=gms_v84 ida=0x5b54cc
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListFailed version=gms_v87 ida=0x5d50cf
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListFailed version=gms_v95 ida=0x576230
func TestMtsResultLoadWishSaleListFailed(t *testing.T) {
	input := NewMtsResultLoadWishSaleListFailed()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2E} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultLoadWishSaleListFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishDone version=gms_v83 ida=0x5a5011
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishDone version=gms_v84 ida=0x5b5501
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishDone version=gms_v87 ida=0x5d5104
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishDone version=gms_v95 ida=0x576270
func TestMtsResultBuyWishDone(t *testing.T) {
	input := NewMtsResultBuyWishDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2F} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyWishDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishFailed version=gms_v83 ida=0x5a503c
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishFailed version=gms_v84 ida=0x5b552c
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishFailed version=gms_v87 ida=0x5d512f
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishFailed version=gms_v95 ida=0x5762a0
func TestMtsResultBuyWishFailed(t *testing.T) {
	input := NewMtsResultBuyWishFailed()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x30} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyWishFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishDone version=gms_v83 ida=0x5a5071
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishDone version=gms_v84 ida=0x5b5561
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishDone version=gms_v87 ida=0x5d5164
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishDone version=gms_v95 ida=0x5762e0
func TestMtsResultCancelWishDone(t *testing.T) {
	input := NewMtsResultCancelWishDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x31} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultCancelWishDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishFailed version=gms_v83 ida=0x5a50df
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishFailed version=gms_v84 ida=0x5b5596
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishFailed version=gms_v87 ida=0x5d5199
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishFailed version=gms_v95 ida=0x576320
func TestMtsResultCancelWishFailed(t *testing.T) {
	input := NewMtsResultCancelWishFailed()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x32} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultCancelWishFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemDone version=gms_v83 ida=0x5a5114
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemDone version=gms_v84 ida=0x5b55cb
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemDone version=gms_v87 ida=0x5d51ce
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemDone version=gms_v95 ida=0x576360
func TestMtsResultBuyItemDone(t *testing.T) {
	input := NewMtsResultBuyItemDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x33} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyItemDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemFailed version=gms_v83 ida=0x5a513f
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemFailed version=gms_v84 ida=0x5b55f6
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemFailed version=gms_v87 ida=0x5d51f9
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemFailed version=gms_v95 ida=0x576390
func TestMtsResultBuyItemFailed(t *testing.T) {
	input := NewMtsResultBuyItemFailed()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x34} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyItemFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemDone version=gms_v83 ida=0x5a5174
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemDone version=gms_v84 ida=0x5b562b
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemDone version=gms_v87 ida=0x5d522e
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemDone version=gms_v95 ida=0x5763d0
func TestMtsResultBuyZzimItemDone(t *testing.T) {
	input := NewMtsResultBuyZzimItemDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x35} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyZzimItemDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemFailed version=gms_v83 ida=0x5a519f
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemFailed version=gms_v84 ida=0x5b5656
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemFailed version=gms_v87 ida=0x5d5259
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemFailed version=gms_v95 ida=0x576400
func TestMtsResultBuyZzimItemFailed(t *testing.T) {
	input := NewMtsResultBuyZzimItemFailed()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x36} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyZzimItemFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemDone version=gms_v83 ida=0x5a51d4
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemDone version=gms_v84 ida=0x5b568b
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemDone version=gms_v87 ida=0x5d528e
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemDone version=gms_v95 ida=0x576440
func TestMtsResultRegisterWishItemDone(t *testing.T) {
	input := NewMtsResultRegisterWishItemDone()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x37} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultRegisterWishItemDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemFailed version=gms_v83 ida=0x5a5209
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemFailed version=gms_v84 ida=0x5b56c0
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemFailed version=gms_v87 ida=0x5d52c3
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemFailed version=gms_v95 ida=0x576480
func TestMtsResultRegisterWishItemFailed(t *testing.T) {
	input := NewMtsResultRegisterWishItemFailed()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x38} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultRegisterWishItemFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBidAuctionFailed version=gms_v83 ida=0x5a5444
// packet-audit:verify packet=field/clientbound/FieldMtsResultBidAuctionFailed version=gms_v84 ida=0x5b58fb
// packet-audit:verify packet=field/clientbound/FieldMtsResultBidAuctionFailed version=gms_v87 ida=0x5d54fe
// packet-audit:verify packet=field/clientbound/FieldMtsResultBidAuctionFailed version=gms_v95 ida=0x5764c0
func TestMtsResultBidAuctionFailed(t *testing.T) {
	input := NewMtsResultBidAuctionFailed()
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x3C} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBidAuctionFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
