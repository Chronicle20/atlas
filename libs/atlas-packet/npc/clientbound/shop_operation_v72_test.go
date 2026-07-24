package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v72 CONFIRM_SHOP_TRANSACTION (CShopDlg::OnPacket @0x6a912b, op 272;
// GMS_v72.1_U_DEVM.exe port 13339). The op-272 arm reads Decode1(mode); every
// notice mode maps to a StringPool notice with no further reads, EXCEPT the
// sole data-bearing arm at mode 14 which reads Decode1(flag) and, when flag!=0,
// DecodeStr(reason). There is NO Decode4 anywhere in the function, so OVER/
// UNDER_LEVEL_REQUIREMENT are version-absent in v72 (no v72 marker — n-a, same
// as v79). GENERIC_ERROR and GENERIC_ERROR_WITH_REASON share mode 14. The mode
// table is byte-identical to the verified v79 read order and matches the v72
// tenant template operations map (OK=0..TRADE_LIMIT=16, GENERIC_ERROR=14):
//
//	0 OK(tab update) | 1/5/9 OUT_OF_STOCK* (StringPool 853) | 2/10 NOT_ENOUGH_
//	MONEY* (5092) | 3 INVENTORY_FULL (854) | 13 NEED_MORE_ITEMS (3953) |
//	16 TRADE_LIMIT (856 default) | 14 GENERIC_ERROR flag=0 (856) /
//	GENERIC_ERROR_WITH_REASON flag!=0 (DecodeStr).
//
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOk version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationInventoryFull version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock2 version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock3 version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney2 version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNeedMoreItems version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationTradeLimit version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v72 ida=0x6a912b
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericErrorWithReason version=gms_v72 ida=0x6a912b
func TestShopOperationArmsV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)

	// Mode-only notice arms: a single mode byte (Decode1(mode) only in v72).
	noticeArms := []struct {
		name string
		body []byte
		mode byte
	}{
		{"Ok", NewShopOperationOk(0x00).Encode(l, ctx)(nil), 0x00},
		{"OutOfStock", NewShopOperationOutOfStock(0x01).Encode(l, ctx)(nil), 0x01},
		{"NotEnoughMoney", NewShopOperationNotEnoughMoney(0x02).Encode(l, ctx)(nil), 0x02},
		{"InventoryFull", NewShopOperationInventoryFull(0x03).Encode(l, ctx)(nil), 0x03},
		{"OutOfStock2", NewShopOperationOutOfStock2(0x05).Encode(l, ctx)(nil), 0x05},
		{"OutOfStock3", NewShopOperationOutOfStock3(0x09).Encode(l, ctx)(nil), 0x09},
		{"NotEnoughMoney2", NewShopOperationNotEnoughMoney2(0x0A).Encode(l, ctx)(nil), 0x0A},
		{"NeedMoreItems", NewShopOperationNeedMoreItems(0x0D).Encode(l, ctx)(nil), 0x0D},
		{"TradeLimit", NewShopOperationTradeLimit(0x10).Encode(l, ctx)(nil), 0x10},
	}
	for _, a := range noticeArms {
		if want := []byte{a.mode}; !bytes.Equal(a.body, want) {
			t.Fatalf("%s v72 body: got % x, want % x", a.name, a.body, want)
		}
	}

	// GENERIC_ERROR arm = v72 mode 14 (0x0E); hasReason=false, no string
	// (mode 14 -> Decode1(flag); flag==0 falls through to a mode-only notice).
	if got := NewShopOperationGenericError(0x0E).Encode(l, ctx)(nil); !bytes.Equal(got, []byte{0x0E, 0x00}) {
		t.Fatalf("GenericError v72 body: got % x, want 0e 00", got)
	}

	// GENERIC_ERROR_WITH_REASON arm = v72 mode 14 (0x0E); flag=1 + DecodeStr(reason).
	wantWR := []byte{0x0E, 0x01, 0x0A, 0x00, 't', 'e', 's', 't', ' ', 'e', 'r', 'r', 'o', 'r'}
	if got := NewShopOperationGenericErrorWithReason(0x0E, "test error").Encode(l, ctx)(nil); !bytes.Equal(got, wantWR) {
		t.Fatalf("GenericErrorWithReason v72 body: got % x, want % x", got, wantWR)
	}
}
