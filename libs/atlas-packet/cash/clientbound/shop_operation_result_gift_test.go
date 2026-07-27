package clientbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte-fixture verify markers are added in Wave 2 once evidence is pinned.

// Per-version dispatcher mode bytes for the gift/coupon/package item-blob arm
// family (task-183 Wave 1.3), taken from
// docs/tasks/task-183-cashshop-result-family/arm-catalog.md (MODERN-5 scope:
// gms_v83/v84/v87/v95, jms_v185 only — legacy modes are Wave 3).

var giftDoneModes = map[string]byte{
	"GMS/v83": 0x5E, "GMS/v84": 0x61, "GMS/v87": 0x63, "GMS/v95": 0x6B, "JMS/v185": 0x5F,
}

// packet-audit:verify packet=cash/clientbound/CashGiftDone version=gms_v83 ida=0x47a856
// packet-audit:verify packet=cash/clientbound/CashGiftDone version=gms_v84 ida=0x47d9f4
// packet-audit:verify packet=cash/clientbound/CashGiftDone version=gms_v87 ida=0x48600e
// packet-audit:verify packet=cash/clientbound/CashGiftDone version=gms_v95 ida=0x497050
// packet-audit:verify packet=cash/clientbound/CashGiftDone version=jms_v185 ida=0x48d3ce
func TestGiftDoneByteFixture(t *testing.T) {
	const recipientName = "Bob"
	itemId := int32(5000000)
	quantity := uint16(1)
	nxCashSpent := int32(1500)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := giftDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no GIFT_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewGiftDone(mode, recipientName, itemId, quantity, nxCashSpent)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode}
			want = append(want, byte(len(recipientName)), 0)
			want = append(want, []byte(recipientName)...)
			want = append(want, byte(itemId), byte(itemId>>8), byte(itemId>>16), byte(itemId>>24))
			want = append(want, byte(quantity), byte(quantity>>8))
			// nxCashSpent:Decode4 is GMS-only on the wire; jms's resolved
			// handler stops after quantity (arm-catalog.md divergence §2).
			wantNxCashSpent := int32(0)
			if v.Region == "GMS" {
				want = append(want, byte(nxCashSpent), byte(nxCashSpent>>8), byte(nxCashSpent>>16), byte(nxCashSpent>>24))
				wantNxCashSpent = nxCashSpent
			}
			if !bytesEqual(got, want) {
				t.Errorf("GIFT_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := GiftDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.RecipientName() != recipientName || output.ItemId() != itemId || output.Quantity() != quantity || output.NxCashSpent() != wantNxCashSpent {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var loadGiftDoneModes = map[string]byte{
	"GMS/v83": 0x4D, "GMS/v84": 0x50, "GMS/v87": 0x52, "GMS/v95": 0x5A, "JMS/v185": 0x50,
}

// giftListEntryBytes reproduces the 98-byte GW_GiftList wire layout:
// liSN(i64,0) nItemID(i32,8) sBuyCharacterName(char[13],12) sText(char[73],25).
func giftListEntryBytes(g GiftListEntry) []byte {
	out := make([]byte, 0, 98)
	sn := uint64(g.SN)
	for i := 0; i < 8; i++ {
		out = append(out, byte(sn>>(8*i)))
	}
	id := uint32(g.ItemId)
	for i := 0; i < 4; i++ {
		out = append(out, byte(id>>(8*i)))
	}
	name := make([]byte, 13)
	copy(name, g.BuyCharacterName)
	out = append(out, name...)
	text := make([]byte, 73)
	copy(text, g.Text)
	out = append(out, text...)
	return out
}

// packet-audit:verify packet=cash/clientbound/CashLoadGiftDone version=gms_v83 ida=0x47959e
// packet-audit:verify packet=cash/clientbound/CashLoadGiftDone version=gms_v84 ida=0x47c73c
// packet-audit:verify packet=cash/clientbound/CashLoadGiftDone version=gms_v87 ida=0x484cc5
// packet-audit:verify packet=cash/clientbound/CashLoadGiftDone version=gms_v95 ida=0x496520
// packet-audit:verify packet=cash/clientbound/CashLoadGiftDone version=jms_v185 ida=0x48bdc8
func TestLoadGiftDoneByteFixture(t *testing.T) {
	gifts := []GiftListEntry{
		{SN: 123456789, ItemId: 5000000, BuyCharacterName: "Alice", Text: "Happy birthday!"},
		{SN: 987654321, ItemId: 5000001, BuyCharacterName: "Carol", Text: "Enjoy"},
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := loadGiftDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no LOAD_GIFT_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewLoadGiftDone(mode, gifts)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, byte(len(gifts)), 0}
			for _, g := range gifts {
				want = append(want, giftListEntryBytes(g)...)
			}
			if !bytesEqual(got, want) {
				t.Errorf("LOAD_GIFT_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := LoadGiftDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || len(output.Gifts()) != len(gifts) {
				t.Fatalf("round-trip mismatch: mode %v gifts %d", output.Mode(), len(output.Gifts()))
			}
			for i, g := range output.Gifts() {
				if g != gifts[i] {
					t.Errorf("gift[%d]: got %+v, want %+v", i, g, gifts[i])
				}
			}
		})
	}
}

var coupleDoneModes = map[string]byte{
	"GMS/v83": 0x87, "GMS/v84": 0x8A, "GMS/v87": 0x8C, "GMS/v95": 0x98, "JMS/v185": 0x8A,
}

// packet-audit:verify packet=cash/clientbound/CashCoupleDone version=gms_v83 ida=0x47b78e
// packet-audit:verify packet=cash/clientbound/CashCoupleDone version=gms_v84 ida=0x47e92c
// packet-audit:verify packet=cash/clientbound/CashCoupleDone version=gms_v87 ida=0x486f6f
// packet-audit:verify packet=cash/clientbound/CashCoupleDone version=gms_v95 ida=0x497b70
// packet-audit:verify packet=cash/clientbound/CashCoupleDone version=jms_v185 ida=0x48e36b
func TestCoupleDoneByteFixture(t *testing.T) {
	item := CashInventoryItem{
		CashId: 111, AccountId: 1, CharacterId: 2, TemplateId: 5390000,
		CommodityId: 100, Quantity: 1, GiftFrom: "", Expiration: 0,
	}
	const recipientName = "Dave"
	itemId := int32(5390000)
	quantity := uint16(1)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := coupleDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no COUPLE_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCoupleDone(mode, item, recipientName, itemId, quantity)
			got := pt.Encode(t, ctx, input.Encode, nil)
			l, _ := testlog.NewNullLogger()
			want := []byte{mode}
			want = append(want, item.EncodeBytes(l)...)
			want = append(want, byte(len(recipientName)), 0)
			want = append(want, []byte(recipientName)...)
			want = append(want, byte(itemId), byte(itemId>>8), byte(itemId>>16), byte(itemId>>24))
			want = append(want, byte(quantity), byte(quantity>>8))
			if !bytesEqual(got, want) {
				t.Errorf("COUPLE_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := CoupleDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.RecipientName() != recipientName || output.ItemId() != itemId || output.Quantity() != quantity {
				t.Errorf("round-trip mismatch: %+v", output)
			}
			if output.Item().TemplateId != item.TemplateId || output.Item().CashId != item.CashId {
				t.Errorf("item round-trip mismatch: got %+v, want %+v", output.Item(), item)
			}
		})
	}
}

// le16/le32 append a little-endian encoded value to a byte slice; shared by
// the remaining arm byte-fixtures in this file.
func le16(b []byte, v uint16) []byte { return append(b, byte(v), byte(v>>8)) }

func le32(b []byte, v int32) []byte {
	u := uint32(v)
	return append(b, byte(u), byte(u>>8), byte(u>>16), byte(u>>24))
}

// asciiStr appends a WriteAsciiString-encoded string (uint16 length + bytes).
func asciiStr(b []byte, s string) []byte {
	b = le16(b, uint16(len(s)))
	return append(b, []byte(s)...)
}

// packedRefBytes reproduces the 8-byte packed record: quantity(u16,0)
// slotPos(u16,2) itemId(i32,4).
func packedRefBytes(r PackedCashItemRef) []byte {
	out := le16(nil, r.Quantity)
	out = le16(out, r.SlotPos)
	out = le32(out, r.ItemId)
	return out
}

var useCouponDoneModes = map[string]byte{
	"GMS/v83": 0x59, "GMS/v84": 0x5C, "GMS/v87": 0x5E, "GMS/v95": 0x66, "JMS/v185": 0x5A,
}

// packet-audit:verify packet=cash/clientbound/CashUseCouponDone version=gms_v83 ida=0x479d8a
// packet-audit:verify packet=cash/clientbound/CashUseCouponDone version=gms_v84 ida=0x47cf28
// packet-audit:verify packet=cash/clientbound/CashUseCouponDone version=gms_v87 ida=0x485563
// packet-audit:verify packet=cash/clientbound/CashUseCouponDone version=gms_v95 ida=0x498670
// packet-audit:verify packet=cash/clientbound/CashUseCouponDone version=jms_v185 ida=0x48c966
func TestUseCouponDoneByteFixture(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	items := []CashInventoryItem{
		{CashId: 1, AccountId: 1, CharacterId: 2, TemplateId: 5000000, CommodityId: 1, Quantity: 1},
	}
	maplePoint := int32(0)
	refs := []PackedCashItemRef{
		{Quantity: 1, SlotPos: 3, ItemId: 5000001},
		{Quantity: 2, SlotPos: 4, ItemId: 5000002},
	}
	meso := int32(0)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := useCouponDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no USE_COUPON_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewUseCouponDone(mode, items, maplePoint, refs, meso)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, byte(len(items))}
			for _, item := range items {
				want = append(want, item.EncodeBytes(l)...)
			}
			want = le32(want, maplePoint)
			want = le32(want, int32(len(refs)))
			for _, ref := range refs {
				want = append(want, packedRefBytes(ref)...)
			}
			want = le32(want, meso)
			if !bytesEqual(got, want) {
				t.Errorf("USE_COUPON_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := UseCouponDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || len(output.Items()) != len(items) || output.MaplePoint() != maplePoint || len(output.Refs()) != len(refs) || output.Meso() != meso {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var giftCouponDoneModes = map[string]byte{
	"GMS/v83": 0x5B, "GMS/v84": 0x5E, "GMS/v87": 0x60, "GMS/v95": 0x68, "JMS/v185": 0x5C,
}

// packet-audit:verify packet=cash/clientbound/CashGiftCouponDone version=gms_v83 ida=0x47a362
// packet-audit:verify packet=cash/clientbound/CashGiftCouponDone version=gms_v84 ida=0x47d500
// packet-audit:verify packet=cash/clientbound/CashGiftCouponDone version=gms_v87 ida=0x485b2c
// packet-audit:verify packet=cash/clientbound/CashGiftCouponDone version=gms_v95 ida=0x498e10
// packet-audit:verify packet=cash/clientbound/CashGiftCouponDone version=jms_v185 ida=0x48cf31
func TestGiftCouponDoneByteFixture(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	const recipientName = "Eve"
	items := []CashInventoryItem{
		{CashId: 2, AccountId: 1, CharacterId: 2, TemplateId: 5000003, CommodityId: 1, Quantity: 1},
	}
	maplePoint := int32(500)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := giftCouponDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no GIFT_COUPON_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewGiftCouponDone(mode, recipientName, items, maplePoint)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode}
			want = asciiStr(want, recipientName)
			want = append(want, byte(len(items)))
			for _, item := range items {
				want = append(want, item.EncodeBytes(l)...)
			}
			want = le32(want, maplePoint)
			if !bytesEqual(got, want) {
				t.Errorf("GIFT_COUPON_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := GiftCouponDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.RecipientName() != recipientName || len(output.Items()) != len(items) || output.MaplePoint() != maplePoint {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var buyPackageDoneModes = map[string]byte{
	"GMS/v83": 0x89, "GMS/v84": 0x8C, "GMS/v87": 0x8E, "GMS/v95": 0x9A, "JMS/v185": 0x8C,
}

// packet-audit:verify packet=cash/clientbound/CashBuyPackageDone version=gms_v83 ida=0x479a1b
// packet-audit:verify packet=cash/clientbound/CashBuyPackageDone version=gms_v84 ida=0x47cbb9
// packet-audit:verify packet=cash/clientbound/CashBuyPackageDone version=gms_v87 ida=0x4851f4
// packet-audit:verify packet=cash/clientbound/CashBuyPackageDone version=gms_v95 ida=0x496b60
// packet-audit:verify packet=cash/clientbound/CashBuyPackageDone version=jms_v185 ida=0x48c593
func TestBuyPackageDoneByteFixture(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	items := []CashInventoryItem{
		{CashId: 3, AccountId: 1, CharacterId: 2, TemplateId: 5000004, CommodityId: 1, Quantity: 1},
		{CashId: 4, AccountId: 1, CharacterId: 2, TemplateId: 5000005, CommodityId: 1, Quantity: 1},
	}
	trailingCount := uint16(0)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := buyPackageDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no BUY_PACKAGE_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuyPackageDone(mode, items, trailingCount)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode, byte(len(items))}
			for _, item := range items {
				want = append(want, item.EncodeBytes(l)...)
			}
			want = le16(want, trailingCount)
			if !bytesEqual(got, want) {
				t.Errorf("BUY_PACKAGE_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := BuyPackageDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || len(output.Items()) != len(items) || output.TrailingCount() != trailingCount {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var giftPackageDoneModes = map[string]byte{
	"GMS/v83": 0x8B, "GMS/v84": 0x8E, "GMS/v87": 0x90, "GMS/v95": 0x9C, "JMS/v185": 0x8E,
}

// packet-audit:verify packet=cash/clientbound/CashGiftPackageDone version=gms_v83 ida=0x479c22
// packet-audit:verify packet=cash/clientbound/CashGiftPackageDone version=gms_v84 ida=0x47cdc0
// packet-audit:verify packet=cash/clientbound/CashGiftPackageDone version=gms_v87 ida=0x4853fb
// packet-audit:verify packet=cash/clientbound/CashGiftPackageDone version=gms_v95 ida=0x496dc0
// packet-audit:verify packet=cash/clientbound/CashGiftPackageDone version=jms_v185 ida=0x48c79a
func TestGiftPackageDoneByteFixture(t *testing.T) {
	const recipientName = "Frank"
	packageId := int32(9000000)
	unused1 := uint16(0)
	unused2 := uint16(0)
	nxCashSpent := int32(3300)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := giftPackageDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no GIFT_PACKAGE_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewGiftPackageDone(mode, recipientName, packageId, unused1, unused2, nxCashSpent)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode}
			want = asciiStr(want, recipientName)
			want = le32(want, packageId)
			want = le16(want, unused1)
			want = le16(want, unused2)
			// nxCashSpent:Decode4 is GMS-only on the wire; jms's resolved
			// handler stops after unused2 (arm-catalog.md divergence §2).
			wantNxCashSpent := int32(0)
			if v.Region == "GMS" {
				want = le32(want, nxCashSpent)
				wantNxCashSpent = nxCashSpent
			}
			if !bytesEqual(got, want) {
				t.Errorf("GIFT_PACKAGE_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := GiftPackageDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.RecipientName() != recipientName || output.PackageId() != packageId || output.Unused1() != unused1 || output.Unused2() != unused2 || output.NxCashSpent() != wantNxCashSpent {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var buyNormalDoneModes = map[string]byte{
	"GMS/v83": 0x8D, "GMS/v84": 0x90, "GMS/v87": 0x92, "GMS/v95": 0x9E, "JMS/v185": 0x90,
}

// packet-audit:verify packet=cash/clientbound/CashBuyNormalDone version=gms_v83 ida=0x47b603
// packet-audit:verify packet=cash/clientbound/CashBuyNormalDone version=gms_v84 ida=0x47e7a1
// packet-audit:verify packet=cash/clientbound/CashBuyNormalDone version=gms_v87 ida=0x486de1
// packet-audit:verify packet=cash/clientbound/CashBuyNormalDone version=gms_v95 ida=0x495310
// packet-audit:verify packet=cash/clientbound/CashBuyNormalDone version=jms_v185 ida=0x48e1dd
func TestBuyNormalDoneByteFixture(t *testing.T) {
	refs := []PackedCashItemRef{
		{Quantity: 1, SlotPos: 5, ItemId: 5000006},
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := buyNormalDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no BUY_NORMAL_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuyNormalDone(mode, refs)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode}
			want = le32(want, int32(len(refs)))
			for _, ref := range refs {
				want = append(want, packedRefBytes(ref)...)
			}
			if !bytesEqual(got, want) {
				t.Errorf("BUY_NORMAL_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := BuyNormalDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || len(output.Refs()) != len(refs) {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var friendshipDoneModes = map[string]byte{
	"GMS/v83": 0x91, "GMS/v84": 0x94, "GMS/v87": 0x96, "GMS/v95": 0xA2, "JMS/v185": 0x94,
}

// packet-audit:verify packet=cash/clientbound/CashFriendshipDone version=gms_v83 ida=0x47b93c
// packet-audit:verify packet=cash/clientbound/CashFriendshipDone version=gms_v84 ida=0x47eada
// packet-audit:verify packet=cash/clientbound/CashFriendshipDone version=gms_v87 ida=0x48711d
// packet-audit:verify packet=cash/clientbound/CashFriendshipDone version=gms_v95 ida=0x497d90
// packet-audit:verify packet=cash/clientbound/CashFriendshipDone version=jms_v185 ida=0x48e51a
func TestFriendshipDoneByteFixture(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	item := CashInventoryItem{CashId: 5, AccountId: 1, CharacterId: 2, TemplateId: 5000007, CommodityId: 1, Quantity: 1}
	const recipientName = "Grace"
	itemId := int32(5000007)
	quantity := uint16(1)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := friendshipDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no FRIENDSHIP_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewFriendshipDone(mode, item, recipientName, itemId, quantity)
			got := pt.Encode(t, ctx, input.Encode, nil)
			want := []byte{mode}
			want = append(want, item.EncodeBytes(l)...)
			want = asciiStr(want, recipientName)
			want = le32(want, itemId)
			want = le16(want, quantity)
			if !bytesEqual(got, want) {
				t.Errorf("FRIENDSHIP_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := FriendshipDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.RecipientName() != recipientName || output.ItemId() != itemId || output.Quantity() != quantity {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}

var rebateDoneModes = map[string]byte{
	"GMS/v83": 0x85, "GMS/v84": 0x88, "GMS/v87": 0x8A, "GMS/v95": 0x96, "JMS/v185": 0x88,
}

// packet-audit:verify packet=cash/clientbound/CashRebateDone version=gms_v83 ida=0x47b4e4
// packet-audit:verify packet=cash/clientbound/CashRebateDone version=gms_v84 ida=0x47e682
// packet-audit:verify packet=cash/clientbound/CashRebateDone version=gms_v87 ida=0x486cc2
// packet-audit:verify packet=cash/clientbound/CashRebateDone version=gms_v95 ida=0x497980
// packet-audit:verify packet=cash/clientbound/CashRebateDone version=jms_v185 ida=0x48e0be
func TestRebateDoneByteFixture(t *testing.T) {
	sn := int64(123456789012345)
	amount := int32(750)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := rebateDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no REBATE_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewRebateDone(mode, sn, amount)
			got := pt.Encode(t, ctx, input.Encode, nil)
			usn := uint64(sn)
			want := []byte{mode}
			for i := 0; i < 8; i++ {
				want = append(want, byte(usn>>(8*i)))
			}
			want = le32(want, amount)
			if !bytesEqual(got, want) {
				t.Errorf("REBATE_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := RebateDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.SN() != sn || output.Amount() != amount {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}
