package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v48 CONFIRM_SHOP_TRANSACTION (CShopDlg::OnPacket@0x5b7a38, nType==230;
// GMS_v48_1_DEVM.exe). The Decode1(mode) result is tested via a chain of
// subtractions rather than a literal switch in the decompiler's rendering;
// derivation of each mode's target op is in
// docs/tasks/task-221-miumiu-travel-store/gms48-shop-operations.md. Modes
// 0/1/2/3/5/9/10/13/16 are fixed-notice, mode-only arms (byte-identical shape
// to every later version). GENERIC_ERROR and GENERIC_ERROR_WITH_REASON share
// mode 14 (Decode1(mode)+Decode1(hasReason)[+DecodeStr(reason)]) — the same
// shape v83 places at mode 17. OVER_LEVEL_REQUIREMENT/UNDER_LEVEL_REQUIREMENT
// are version-absent (no Decode4-driven numeric-level arm anywhere in the
// function) — no v48 marker for either, matching v61/v72/v79's precedent.
//
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOk version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationInventoryFull version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock2 version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock3 version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney2 version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNeedMoreItems version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationTradeLimit version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v48 ida=0x5b7a38
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericErrorWithReason version=gms_v48 ida=0x5b7a38
func TestShopOperationArmsV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)

	// Mode-only notice arms: a single mode byte.
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
			t.Fatalf("%s v48 body: got % x, want % x", a.name, a.body, want)
		}
	}

	// GENERIC_ERROR arm = v48 mode 14 (0x0E); hasReason=false, no string.
	if got := NewShopOperationGenericError(0x0E).Encode(l, ctx)(nil); !bytes.Equal(got, []byte{0x0E, 0x00}) {
		t.Fatalf("GenericError v48 body: got % x, want 0e 00", got)
	}

	// GENERIC_ERROR_WITH_REASON arm = v48 mode 14 (0x0E); flag=1 + DecodeStr(reason).
	wantWR := []byte{0x0E, 0x01, 0x0A, 0x00, 't', 'e', 's', 't', ' ', 'e', 'r', 'r', 'o', 'r'}
	if got := NewShopOperationGenericErrorWithReason(0x0E, "test error").Encode(l, ctx)(nil); !bytes.Equal(got, wantWR) {
		t.Fatalf("GenericErrorWithReason v48 body: got % x, want % x", got, wantWR)
	}
}
