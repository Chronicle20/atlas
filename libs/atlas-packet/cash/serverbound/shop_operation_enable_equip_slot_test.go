package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationEnableEquipSlot version=gms_v95 ida=0x48e130
// packet-audit:verify packet=cash/serverbound/CashShopOperationEnableEquipSlot version=gms_v87 ida=0x476766
// packet-audit:verify packet=cash/serverbound/CashShopOperationEnableEquipSlot version=gms_v83 ida=0x46c8e1
// packet-audit:verify packet=cash/serverbound/CashShopOperationEnableEquipSlot version=jms_v185 ida=0x47cc5e
// packet-audit:verify packet=cash/serverbound/CashShopOperationEnableEquipSlot version=gms_v84 ida=0x46efb6
func TestShopOperationEnableEquipSlotRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationEnableEquipSlot{pointType: true, currency: 2, flag: 1, serialNumber: 12345}
			output := ShopOperationEnableEquipSlot{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PointType() != input.PointType() {
				t.Errorf("pointType: got %v, want %v", output.PointType(), input.PointType())
			}
			// Legacy GMS (<83, e.g. v79) carries currency + a constant flag byte
			// between pointType and serialNumber; v83+ omits both.
			if v.Region == "GMS" && v.MajorVersion < 83 {
				if output.Currency() != input.Currency() {
					t.Errorf("currency: got %v, want %v", output.Currency(), input.Currency())
				}
				if output.Flag() != input.Flag() {
					t.Errorf("flag: got %v, want %v", output.Flag(), input.Flag())
				}
			} else {
				if output.Currency() != 0 {
					t.Errorf("currency: got %v, want 0 for %s", output.Currency(), v.Name)
				}
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}

// TestShopOperationEnableEquipSlotV79Bytes pins the v79 legacy body. IDA v79
// CCashShop::OnEnableEquipSlotExt@0x469fa9: COutPacket(221) Encode1(6|7)=mode
// (routed op) then Encode1(v45==2)=pointType, Encode4(v45)=currency, Encode1(1)=
// constant flag, Encode4(a2)=serialNumber. Body after the mode byte is
// pointType(1)+currency(4)+flag(1)+serialNumber(4).
// packet-audit:verify packet=cash/serverbound/CashShopOperationEnableEquipSlot version=gms_v79 ida=0x469fa9
func TestShopOperationEnableEquipSlotV79Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationEnableEquipSlot{pointType: true, currency: 0x01020304, flag: 1, serialNumber: 0x05060708}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	if got != "01"+"04030201"+"01"+"08070605" {
		t.Errorf("v79 bytes: got %s, want 01040302010108070605", got)
	}
}

// TestShopOperationEnableEquipSlotV72Bytes pins the v72 legacy body. The v72
// send is CCashShop @0x468e43. Reconciled 2026-08-19 against a direct
// re-decompile (task-240 task 24): the raw linked symbol (func_query /
// lookup_funcs against the name table, not the decompiler's pretty-printed
// header, which stalely displays "OnIncCharacterSlotCount") is
// ?OnBuySlotInc@CCashShop@@QAEXJ@Z, size 0x407 — a single pre-split handler
// covering all cash-shop slot-purchase tabs (equip/use/setup/etc/character).
// v79 splits this into three distinct functions: OnBuySlotInc (0x466b13,
// 0x359, itemType-gated generic tabs, mode=6 constant, no currency field),
// OnIncCharacterSlotCount (0x4673be, 0x21d, GetEquipExtItem +
// CConfirmPurchaseDlg::Confirm, mode=9 constant, no currency field), and
// OnEnableEquipSlotExt (0x469fa9, 0x407). v72's 0x468e43 matches
// OnEnableEquipSlotExt byte-for-byte in size and body shape (same StringPool
// ids 558/537/493/494/508, same GetSlotIncDelta/48-96 slot-cap logic, same
// CS_COMMODITY_EX construction) — NOT the structurally divergent v79
// OnIncCharacterSlotCount. v72 also has a generic OnCashItemResIncSlotCountDone
// /Failed response pair (0x472686/0x47277a) but no distinctly-named
// OnCashItemResEnableEquipSlotExtDone, consistent with v72 not yet having
// split the response side either. Body: COutPacket(219)
// Encode1((v/1000==9110)+6)=mode 6|7 (routed op) @0x469184 then
// Encode1(v45==2)=pointType @0x469194, Encode4(v45)=currency @0x46919f,
// Encode1(1)=constant flag @0x4691a8, Encode4(a2)=serialNumber @0x4691b3.
// Body after the mode byte is pointType(1)+currency(4)+flag(1)+serialNumber(4)
// — byte-identical to the v79 legacy body (both GMS<83). The equip-slot-
// extension send DOES exist on v72.
// packet-audit:verify packet=cash/serverbound/CashShopOperationEnableEquipSlot version=gms_v72 ida=0x468e43
func TestShopOperationEnableEquipSlotV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationEnableEquipSlot{pointType: true, currency: 0x01020304, flag: 1, serialNumber: 0x05060708}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "01"+"04030201"+"01"+"08070605" {
		t.Errorf("v72 bytes: got %s, want 01040302010108070605", got)
	}
}
