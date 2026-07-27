package clientbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify markers are added in Wave 2 once evidence is pinned.

// Per-version dispatcher mode bytes for the scalar/notice arm family
// (task-183 Wave 1.4), taken from
// docs/tasks/task-183-cashshop-result-family/arm-catalog.md (MODERN-5 scope:
// gms_v83/v84/v87/v95, jms_v185 only — legacy modes are Wave 3).

var limitGoodsCountChangedModes = map[string]byte{
	"GMS/v83": 0x47, "GMS/v84": 0x4A, "GMS/v87": 0x4C, "GMS/v95": 0x54, "JMS/v185": 0x4A,
}

func TestLimitGoodsCountChangedByteFixture(t *testing.T) {
	itemId := int32(5390000)
	sn := int32(42)
	remainCount := int32(3)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := limitGoodsCountChangedModes[variantKey(v)]
			if !ok {
				t.Skipf("no LIMIT_GOODS_COUNT_CHANGED mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewLimitGoodsCountChanged(mode, itemId, sn, remainCount)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode}
			want = le32(want, itemId)
			want = le32(want, sn)
			want = le32(want, remainCount)
			if !bytesEqual(got, want) {
				t.Errorf("LIMIT_GOODS_COUNT_CHANGED bytes: got %v, want %v", got, want)
			}
			output := LimitGoodsCountChanged{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.ItemId() != itemId || output.SN() != sn || output.RemainCount() != remainCount {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var destroyDoneModes = map[string]byte{
	"GMS/v83": 0x6C, "GMS/v84": 0x6F, "GMS/v87": 0x71, "GMS/v95": 0x7B, "JMS/v185": 0x6F,
}

// TestDestroyDoneByteFixture is the red->green anchor for the true-scalar
// shape (mode + 8-byte SN) in this file.
func TestDestroyDoneByteFixture(t *testing.T) {
	sn := int64(123456789012345)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := destroyDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no DESTROY_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewDestroyDone(mode, sn)
			got := pt.Encode(t, ctx, input.Encode, nil)
			usn := uint64(sn)
			want := []byte{mode}
			for i := 0; i < 8; i++ {
				want = append(want, byte(usn>>(8*i)))
			}
			if !bytesEqual(got, want) {
				t.Errorf("DESTROY_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := DestroyDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.SN() != sn {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var expireDoneModes = map[string]byte{
	"GMS/v83": 0x6E, "GMS/v84": 0x71, "GMS/v87": 0x73, "GMS/v95": 0x7D, "JMS/v185": 0x71,
}

func TestExpireDoneByteFixture(t *testing.T) {
	sn := int64(987654321098765)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := expireDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no EXPIRE_DONE mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewExpireDone(mode, sn)
			got := pt.Encode(t, ctx, input.Encode, nil)
			usn := uint64(sn)
			want := []byte{mode}
			for i := 0; i < 8; i++ {
				want = append(want, byte(usn>>(8*i)))
			}
			if !bytesEqual(got, want) {
				t.Errorf("EXPIRE_DONE bytes: got %v, want %v", got, want)
			}
			output := ExpireDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.SN() != sn {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var purchaseRecordDoneModes = map[string]byte{
	"GMS/v83": 0x9A, "GMS/v84": 0x9D, "GMS/v87": 0xA3, "GMS/v95": 0xAF, "JMS/v185": 0x9D,
}

func TestPurchaseRecordDoneByteFixture(t *testing.T) {
	goodsSN := int32(778899)
	purchased := byte(1)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := purchaseRecordDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no PURCHASE_RECORD mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewPurchaseRecordDone(mode, goodsSN, purchased)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode}
			want = le32(want, goodsSN)
			want = append(want, purchased)
			if !bytesEqual(got, want) {
				t.Errorf("PURCHASE_RECORD bytes: got %v, want %v", got, want)
			}
			output := PurchaseRecordDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.GoodsSN() != goodsSN || output.Purchased() != purchased {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

// FREE_CASH_ITEM_DONE is present only in v87/v95/jms among MODERN versions
// (n-a v83/v84 per arm-catalog.md §6 "Free-cash-item / purchase-record are
// era-gated, not monotonic").
var freeCashItemDoneModes = map[string]byte{
	"GMS/v87": 0x9E, "GMS/v95": 0xAA, "JMS/v185": 0xA1,
}

// TestFreeCashItemDoneByteFixture is the red->green anchor for the
// item-blob shape (mode + 55-byte GW_CashItemInfo blob) in this file — the
// catalog's coarse "scalar" shape label is wrong for this arm.
func TestFreeCashItemDoneByteFixture(t *testing.T) {
	item := CashInventoryItem{
		CashId: 222, AccountId: 1, CharacterId: 2, TemplateId: 5220000,
		CommodityId: 200, Quantity: 1, GiftFrom: "", Expiration: 0,
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := freeCashItemDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no FREE_CASH_ITEM_DONE mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewFreeCashItemDone(mode, item)
			got := pt.Encode(t, ctx, input.Encode, nil)
			l, _ := testlog.NewNullLogger()
			want := []byte{mode}
			want = append(want, item.EncodeBytes(l)...)
			if !bytesEqual(got, want) {
				t.Errorf("FREE_CASH_ITEM_DONE bytes: got %v, want %v", got, want)
			}
			output := FreeCashItemDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode {
				t.Errorf("round-trip mode mismatch: got %v, want %v", output.Mode(), mode)
			}
			if output.Item().TemplateId != item.TemplateId || output.Item().CashId != item.CashId {
				t.Errorf("item round-trip mismatch: got %+v, want %+v", output.Item(), item)
			}
		})
	}
}
