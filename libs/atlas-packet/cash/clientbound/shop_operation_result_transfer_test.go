package clientbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte-fixture verify markers are added in Wave 2 once evidence is pinned.

// Per-version dispatcher mode bytes for the transfer/name-change/maple-point
// arm family (task-183 Wave 1.4), taken from
// docs/tasks/task-183-cashshop-result-family/arm-catalog.md (MODERN-5 scope:
// gms_v83/v84/v87/v95, jms_v185 only — legacy modes are Wave 3).

var nameChangeBuyDoneModes = map[string]byte{
	"GMS/v83": 0x9E, "GMS/v84": 0xA1, "GMS/v87": 0xA7, "GMS/v95": 0xB3, "JMS/v185": 0xA5,
}

func TestNameChangeBuyDoneByteFixture(t *testing.T) {
	item := CashInventoryItem{
		CashId: 333, AccountId: 1, CharacterId: 2, TemplateId: 5152001,
		CommodityId: 300, Quantity: 1, GiftFrom: "", Expiration: 0,
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := nameChangeBuyDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no NAME_CHANGE_BUY_DONE mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewNameChangeBuyDone(mode, item)
			got := pt.Encode(t, ctx, input.Encode, nil)
			l, _ := testlog.NewNullLogger()
			want := []byte{mode}
			want = append(want, item.EncodeBytes(l)...)
			if !bytesEqual(got, want) {
				t.Errorf("NAME_CHANGE_BUY_DONE bytes: got %v, want %v", got, want)
			}
			output := NameChangeBuyDone{}
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

var transferWorldDoneModes = map[string]byte{
	"GMS/v83": 0xA0, "GMS/v84": 0xA3, "GMS/v87": 0xA9, "GMS/v95": 0xB5, "JMS/v185": 0xAE,
}

func TestTransferWorldDoneByteFixture(t *testing.T) {
	item := CashInventoryItem{
		CashId: 444, AccountId: 1, CharacterId: 2, TemplateId: 5150000,
		CommodityId: 400, Quantity: 1, GiftFrom: "", Expiration: 0,
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := transferWorldDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no TRANSFER_WORLD_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewTransferWorldDone(mode, item)
			got := pt.Encode(t, ctx, input.Encode, nil)
			l, _ := testlog.NewNullLogger()
			want := []byte{mode}
			want = append(want, item.EncodeBytes(l)...)
			if !bytesEqual(got, want) {
				t.Errorf("TRANSFER_WORLD_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := TransferWorldDone{}
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

// CHANGE_MAPLE_POINT_SUCCESS is present only in v84/v87/v95 among MODERN
// versions (n-a v83, n-a jms per arm-catalog.md §4/§5).
var changeMaplePointDoneModes = map[string]byte{
	"GMS/v84": 0xA9, "GMS/v87": 0xAF, "GMS/v95": 0xBB,
}

func TestChangeMaplePointDoneByteFixture(t *testing.T) {
	sn := int64(555666777888999)
	count := int32(50)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			mode, ok := changeMaplePointDoneModes[variantKey(v)]
			if !ok {
				t.Skipf("no CHANGE_MAPLE_POINT_SUCCESS mode byte for %s", variantKey(v))
			}
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewChangeMaplePointDone(mode, sn, count)
			got := pt.Encode(t, ctx, input.Encode, nil)
			usn := uint64(sn)
			want := []byte{mode}
			for i := 0; i < 8; i++ {
				want = append(want, byte(usn>>(8*i)))
			}
			want = le32(want, count)
			if !bytesEqual(got, want) {
				t.Errorf("CHANGE_MAPLE_POINT_SUCCESS bytes: got %v, want %v", got, want)
			}
			output := ChangeMaplePointDone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != mode || output.SN() != sn || output.Count() != count {
				t.Errorf("round-trip mismatch: %+v", output)
			}
		})
	}
}
