package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v79 CONFIRM_SHOP_TRANSACTION (CShopDlg::OnPacket @0x6d6eb9, op 284;
// GMS_v79_1_DEVM.exe port 13340). The v79 switch reads only Decode1(mode) for
// every arm except the sole data-bearing arm at mode 14 (v18==1), which reads
// Decode1(flag) then conditionally DecodeStr(reason). There is NO Decode4
// anywhere in the function, so OVER/UNDER_LEVEL_REQUIREMENT are version-absent
// in v79 (no v79 marker — n-a). GENERIC_ERROR and GENERIC_ERROR_WITH_REASON
// share mode 14 (the same flag+conditional-string shape v83 placed at mode 17).
// The notice modes (0,1,2,3,5,9,10,13,16) are byte-identical to v83. Mode bytes
// come from docs/packets/dispatchers/npc_shop_operation.yaml (gms_v79 column).
//
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOk version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationInventoryFull version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock2 version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock3 version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney2 version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNeedMoreItems version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationTradeLimit version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v79 ida=0x6d6eb9
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericErrorWithReason version=gms_v79 ida=0x6d6eb9
func TestShopOperationArmsV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)

	// Mode-only notice arms: a single mode byte (Decode1(mode) only in v79).
	noticeArms := []struct {
		name string
		body []byte // Encode output
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
			t.Fatalf("%s v79 body: got % x, want % x", a.name, a.body, want)
		}
	}

	// GENERIC_ERROR arm = v79 mode 14 (0x0E); hasReason=false, no string.
	// v79: mode 14 -> Decode1(flag); flag==0 falls through to a mode-only notice.
	if got := NewShopOperationGenericError(0x0E).Encode(l, ctx)(nil); !bytes.Equal(got, []byte{0x0E, 0x00}) {
		t.Fatalf("GenericError v79 body: got % x, want 0e 00", got)
	}

	// GENERIC_ERROR_WITH_REASON arm = v79 mode 14 (0x0E); flag=1 + DecodeStr(reason).
	// EncodeStr("test error") = uint16-LE length 0x0A 0x00 then 10 ASCII bytes.
	wantWR := []byte{0x0E, 0x01, 0x0A, 0x00, 't', 'e', 's', 't', ' ', 'e', 'r', 'r', 'o', 'r'}
	if got := NewShopOperationGenericErrorWithReason(0x0E, "test error").Encode(l, ctx)(nil); !bytes.Equal(got, wantWR) {
		t.Fatalf("GenericErrorWithReason v79 body: got % x, want % x", got, wantWR)
	}
}
