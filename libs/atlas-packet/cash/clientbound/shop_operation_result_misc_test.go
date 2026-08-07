package clientbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte-fixture verify markers are added in Wave 2 once evidence is pinned.

// Per-version dispatcher mode bytes for the scalar/notice arm family
// (task-183 Wave 1.4), taken from
// docs/tasks/task-183-cashshop-result-family/arm-catalog.md (MODERN-5 scope:
// gms_v83/v84/v87/v95, jms_v185 only — legacy modes are Wave 3).

var limitGoodsCountChangedModes = map[string]byte{
	"GMS/v48": 0x29, "GMS/v61": 0x2B, "GMS/v72": 0x33, "GMS/v79": 0x3F,
	"GMS/v83": 0x47, "GMS/v84": 0x4A, "GMS/v87": 0x4C, "GMS/v95": 0x54, "JMS/v185": 0x4A,
}

// packet-audit:verify packet=cash/clientbound/CashLimitGoodsCountChanged version=gms_v48 ida=0x4536d3
// packet-audit:verify packet=cash/clientbound/CashLimitGoodsCountChanged version=gms_v61 ida=0x4612a4
// packet-audit:verify packet=cash/clientbound/CashLimitGoodsCountChanged version=gms_v72 ida=0x470d3e
// packet-audit:verify packet=cash/clientbound/CashLimitGoodsCountChanged version=gms_v79 ida=0x472018
// packet-audit:verify packet=cash/clientbound/CashLimitGoodsCountChanged version=gms_v83 ida=0x47908a
// packet-audit:verify packet=cash/clientbound/CashLimitGoodsCountChanged version=gms_v84 ida=0x47c1bc
// packet-audit:verify packet=cash/clientbound/CashLimitGoodsCountChanged version=gms_v87 ida=0x48471f
// packet-audit:verify packet=cash/clientbound/CashLimitGoodsCountChanged version=gms_v95 ida=0x493f30
// packet-audit:verify packet=cash/clientbound/CashLimitGoodsCountChanged version=jms_v185 ida=0x48b4d0
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
	"GMS/v48": 0x45, "GMS/v61": 0x4E, "GMS/v72": 0x56, "GMS/v79": 0x64,
	"GMS/v83": 0x6C, "GMS/v84": 0x6F, "GMS/v87": 0x71, "GMS/v95": 0x7B, "JMS/v185": 0x6F,
}

// TestDestroyDoneByteFixture is the red->green anchor for the true-scalar
// shape (mode + 8-byte SN) in this file.
// packet-audit:verify packet=cash/clientbound/CashDestroyDone version=gms_v48 ida=0x4553e0
// packet-audit:verify packet=cash/clientbound/CashDestroyDone version=gms_v61 ida=0x463150
// packet-audit:verify packet=cash/clientbound/CashDestroyDone version=gms_v72 ida=0x472e4f
// packet-audit:verify packet=cash/clientbound/CashDestroyDone version=gms_v79 ida=0x47431b
// packet-audit:verify packet=cash/clientbound/CashDestroyDone version=gms_v83 ida=0x47b420
// packet-audit:verify packet=cash/clientbound/CashDestroyDone version=gms_v84 ida=0x47e5be
// packet-audit:verify packet=cash/clientbound/CashDestroyDone version=gms_v87 ida=0x486bfe
// packet-audit:verify packet=cash/clientbound/CashDestroyDone version=gms_v95 ida=0x495250
// packet-audit:verify packet=cash/clientbound/CashDestroyDone version=jms_v185 ida=0x48dffa
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
	"GMS/v48": 0x47, "GMS/v61": 0x50, "GMS/v72": 0x58, "GMS/v79": 0x66,
	"GMS/v83": 0x6E, "GMS/v84": 0x71, "GMS/v87": 0x73, "GMS/v95": 0x7D, "JMS/v185": 0x71,
}

// packet-audit:verify packet=cash/clientbound/CashExpireDone version=gms_v48 ida=0x455191
// packet-audit:verify packet=cash/clientbound/CashExpireDone version=gms_v61 ida=0x462edf
// packet-audit:verify packet=cash/clientbound/CashExpireDone version=gms_v72 ida=0x472bda
// packet-audit:verify packet=cash/clientbound/CashExpireDone version=gms_v79 ida=0x4740a6
// packet-audit:verify packet=cash/clientbound/CashExpireDone version=gms_v83 ida=0x47b1ab
// packet-audit:verify packet=cash/clientbound/CashExpireDone version=gms_v84 ida=0x47e349
// packet-audit:verify packet=cash/clientbound/CashExpireDone version=gms_v87 ida=0x486981
// packet-audit:verify packet=cash/clientbound/CashExpireDone version=gms_v95 ida=0x497760
// packet-audit:verify packet=cash/clientbound/CashExpireDone version=jms_v185 ida=0x48dd84
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

// purchaseRecordDoneModes: n-a in GMS/v48/v61 (feature does not exist yet —
// arm-catalog.md §6). First present starting GMS/v72.
var purchaseRecordDoneModes = map[string]byte{
	"GMS/v72": 0x84, "GMS/v79": 0x92,
	"GMS/v83": 0x9A, "GMS/v84": 0x9D, "GMS/v87": 0xA3, "GMS/v95": 0xAF, "JMS/v185": 0x9D,
}

// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordDone version=gms_v72 ida=0x473ac0
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordDone version=gms_v79 ida=0x474f8c
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordDone version=gms_v83 ida=0x47c091
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordDone version=gms_v84 ida=0x47f22f
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordDone version=gms_v87 ida=0x487872
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordDone version=gms_v95 ida=0x495b50
// packet-audit:verify packet=cash/clientbound/CashPurchaseRecordDone version=jms_v185 ida=0x48e72f
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
// packet-audit:verify packet=cash/clientbound/CashFreeCashItemDone version=gms_v87 ida=0x4850ac
// packet-audit:verify packet=cash/clientbound/CashFreeCashItemDone version=gms_v95 ida=0x494880
// packet-audit:verify packet=cash/clientbound/CashFreeCashItemDone version=jms_v185 ida=0x48c2a9
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
