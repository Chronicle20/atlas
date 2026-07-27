package clientbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify markers are added in Wave 2 once evidence is pinned.

// Per-version dispatcher mode bytes for the gachapon arm family (task-183
// Wave 1.4), taken from docs/tasks/task-183-cashshop-result-family/
// arm-catalog.md. Present only in v84/v87/v95 among MODERN versions (n-a
// v83, n-a jms — verifiably absent from the whole binary in both).

var gachaponOpenDoneModes = map[string]byte{
	"GMS/v84": 0xA5, "GMS/v87": 0xAB, "GMS/v95": 0xB7,
}

// TestGachaponOpenDoneByteFixture is the red->green anchor for the
// conditional-blob shape (mode + scalars + CONDITIONAL 55-byte item-blob +
// trailing scalars) in this file. Exercises both the isCashItem==0 (blob
// omitted) and isCashItem!=0 (blob present) branches.
func TestGachaponOpenDoneByteFixture(t *testing.T) {
	sn := int64(11223344556677)
	remain := int32(2)
	resultCode := int32(0)
	resultParam2 := byte(1)
	item := CashInventoryItem{
		CashId: 555, AccountId: 1, CharacterId: 2, TemplateId: 5390000,
		CommodityId: 500, Quantity: 1, GiftFrom: "", Expiration: 0,
	}
	l, _ := testlog.NewNullLogger()

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := gachaponOpenDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no GACHAPON_OPEN_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			t.Run("isCashItem=0", func(t *testing.T) {
				input := NewGachaponOpenDone(mode, sn, remain, 0, CashInventoryItem{}, resultCode, resultParam2)
				got := pt.Encode(t, ctx, input.Encode, nil)
				usn := uint64(sn)
				want := []byte{mode}
				for i := 0; i < 8; i++ {
					want = append(want, byte(usn>>(8*i)))
				}
				want = le32(want, remain)
				want = append(want, 0)
				want = le32(want, resultCode)
				want = append(want, resultParam2)
				if !bytesEqual(got, want) {
					t.Errorf("GACHAPON_OPEN_SUCCESS (no blob) bytes: got %v, want %v", got, want)
				}
				output := GachaponOpenDone{}
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Mode() != mode || output.SN() != sn || output.Remain() != remain || output.IsCashItem() != 0 || output.ResultCode() != resultCode || output.ResultParam2() != resultParam2 {
					t.Errorf("round-trip mismatch (no blob): %+v", output)
				}
			})

			t.Run("isCashItem=1", func(t *testing.T) {
				input := NewGachaponOpenDone(mode, sn, remain, 1, item, resultCode, resultParam2)
				got := pt.Encode(t, ctx, input.Encode, nil)
				usn := uint64(sn)
				want := []byte{mode}
				for i := 0; i < 8; i++ {
					want = append(want, byte(usn>>(8*i)))
				}
				want = le32(want, remain)
				want = append(want, 1)
				want = append(want, item.EncodeBytes(l)...)
				want = le32(want, resultCode)
				want = append(want, resultParam2)
				if !bytesEqual(got, want) {
					t.Errorf("GACHAPON_OPEN_SUCCESS (with blob) bytes: got %v, want %v", got, want)
				}
				output := GachaponOpenDone{}
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Mode() != mode || output.SN() != sn || output.Remain() != remain || output.IsCashItem() != 1 || output.ResultCode() != resultCode || output.ResultParam2() != resultParam2 {
					t.Errorf("round-trip mismatch (with blob): %+v", output)
				}
				if output.NewItem().TemplateId != item.TemplateId || output.NewItem().CashId != item.CashId {
					t.Errorf("item round-trip mismatch: got %+v, want %+v", output.NewItem(), item)
				}
			})
		})
	}
}

var gachaponCopyDoneModes = map[string]byte{
	"GMS/v84": 0xA7, "GMS/v87": 0xAD, "GMS/v95": 0xB9,
}

// TestGachaponCopyDoneByteFixture exercises the compound-AND gate: the
// item-blob is present iff flag1!=0 AND flag2!=0 — distinct from
// GachaponOpenDone's single-flag gate.
func TestGachaponCopyDoneByteFixture(t *testing.T) {
	unused1 := int32(0)
	unused2 := int32(0)
	lostItemId := int32(5000000)
	lostNumber := int32(1)
	item := CashInventoryItem{
		CashId: 666, AccountId: 1, CharacterId: 2, TemplateId: 5391000,
		CommodityId: 600, Quantity: 1, GiftFrom: "", Expiration: 0,
	}
	l, _ := testlog.NewNullLogger()

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := gachaponCopyDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no GACHAPON_COPY_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			t.Run("flags=0,0", func(t *testing.T) {
				input := NewGachaponCopyDone(mode, 0, 0, unused1, unused2, lostItemId, lostNumber, CashInventoryItem{})
				got := pt.Encode(t, ctx, input.Encode, nil)
				want := []byte{mode, 0, 0}
				want = le32(want, unused1)
				want = le32(want, unused2)
				want = le32(want, lostItemId)
				want = le32(want, lostNumber)
				if !bytesEqual(got, want) {
					t.Errorf("GACHAPON_COPY_SUCCESS (no blob, both flags 0) bytes: got %v, want %v", got, want)
				}
				output := GachaponCopyDone{}
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Flag1() != 0 || output.Flag2() != 0 {
					t.Errorf("round-trip flag mismatch: %+v", output)
				}
			})

			t.Run("flags=1,0", func(t *testing.T) {
				input := NewGachaponCopyDone(mode, 1, 0, unused1, unused2, lostItemId, lostNumber, CashInventoryItem{})
				got := pt.Encode(t, ctx, input.Encode, nil)
				want := []byte{mode, 1, 0}
				want = le32(want, unused1)
				want = le32(want, unused2)
				want = le32(want, lostItemId)
				want = le32(want, lostNumber)
				if !bytesEqual(got, want) {
					t.Errorf("GACHAPON_COPY_SUCCESS (no blob, flag1 only) bytes: got %v, want %v", got, want)
				}
				output := GachaponCopyDone{}
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Flag1() != 1 || output.Flag2() != 0 {
					t.Errorf("round-trip flag mismatch: %+v", output)
				}
			})

			t.Run("flags=1,1", func(t *testing.T) {
				input := NewGachaponCopyDone(mode, 1, 1, unused1, unused2, lostItemId, lostNumber, item)
				got := pt.Encode(t, ctx, input.Encode, nil)
				want := []byte{mode, 1, 1}
				want = le32(want, unused1)
				want = le32(want, unused2)
				want = le32(want, lostItemId)
				want = le32(want, lostNumber)
				want = append(want, item.EncodeBytes(l)...)
				if !bytesEqual(got, want) {
					t.Errorf("GACHAPON_COPY_SUCCESS (with blob) bytes: got %v, want %v", got, want)
				}
				output := GachaponCopyDone{}
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Flag1() != 1 || output.Flag2() != 1 || output.LostItemId() != lostItemId || output.LostNumber() != lostNumber {
					t.Errorf("round-trip mismatch: %+v", output)
				}
				if output.Item().TemplateId != item.TemplateId || output.Item().CashId != item.CashId {
					t.Errorf("item round-trip mismatch: got %+v, want %+v", output.Item(), item)
				}
			})
		})
	}
}
