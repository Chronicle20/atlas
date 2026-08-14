package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v92 CONFIRM_SHOP_TRANSACTION (CShopDlg::OnPacket@0x6e0390,
// GMS_v92_1_DEVM.exe). switch(Decode1(mode)) is case-for-case structurally
// identical to gms_v95's already-verified CShopDlg::OnPacket@0x6eb7d0: mode
// 0 (tab update, no notice), 1/5/9 (fixed notice), 2/10 (fixed notice), 3
// (fixed notice), 4/8 (silent return, no case in Atlas), 13 (fixed notice),
// 14 (Decode4 + Format, negative delta), 15 (Decode4 + Format, positive
// delta), 16 (fixed notice), 17 (fixed notice, no further reads — same
// GENERIC_ERROR mode number used at every prior version per
// docs/packets/dispatchers/npc_shop_operation.yaml), 18 (fixed notice, no
// Atlas struct — untracked at v95 too), 19 (Decode1(hasReason) +
// conditional DecodeStr(reason) — GENERIC_ERROR_WITH_REASON; hasReason=0
// falls through to the SAME default-notice path as GENERIC_ERROR), default
// (fixed notice, GENERIC_ERROR fallback). Mode bytes match the v95 column
// task-14 already wired into template_gms_92_1.json's NPCShopOperation
// (0x165) operations table.
//
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOk version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationInventoryFull version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock2 version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOutOfStock3 version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNotEnoughMoney2 version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationNeedMoreItems version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationTradeLimit version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationOverLevelRequirement version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationUnderLevelRequirement version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericError version=gms_v92 ida=0x6e0390
// packet-audit:verify packet=npc/clientbound/NpcShopOperationGenericErrorWithReason version=gms_v92 ida=0x6e0390
func TestShopOperationArmsV92(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)

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
			t.Fatalf("%s v92 body: got % x, want % x", a.name, a.body, want)
		}
	}

	// OVER/UNDER_LEVEL_REQUIREMENT: mode byte + Int32 level (Decode4 @0x6e0612/0x6e06c1).
	if got := NewShopOperationOverLevelRequirement(0x0E, 5).Encode(l, ctx)(nil); !bytes.Equal(got, []byte{0x0E, 0x05, 0x00, 0x00, 0x00}) {
		t.Fatalf("OverLevelRequirement v92 body: got % x, want 0e 05 00 00 00", got)
	}
	if got := NewShopOperationUnderLevelRequirement(0x0F, 7).Encode(l, ctx)(nil); !bytes.Equal(got, []byte{0x0F, 0x07, 0x00, 0x00, 0x00}) {
		t.Fatalf("UnderLevelRequirement v92 body: got % x, want 0f 07 00 00 00", got)
	}

	// GENERIC_ERROR arm = v92 mode 17 (0x11); hasReason=false, no string.
	if got := NewShopOperationGenericError(0x11).Encode(l, ctx)(nil); !bytes.Equal(got, []byte{0x11, 0x00}) {
		t.Fatalf("GenericError v92 body: got % x, want 11 00", got)
	}

	// GENERIC_ERROR_WITH_REASON arm = v92 mode 19 (0x13); flag=1 + DecodeStr(reason).
	wantWR := []byte{0x13, 0x01, 0x0A, 0x00, 't', 'e', 's', 't', ' ', 'e', 'r', 'r', 'o', 'r'}
	if got := NewShopOperationGenericErrorWithReason(0x13, "test error").Encode(l, ctx)(nil); !bytes.Equal(got, wantWR) {
		t.Fatalf("GenericErrorWithReason v92 body: got % x, want % x", got, wantWR)
	}
}
