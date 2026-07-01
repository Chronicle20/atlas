package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestShopRechargeByteV79 pins the gms_v79 NPC_SHOP RECHARGE body (op byte 2,
// dispatcher prefix; body only here).
//
// IDA: CShopDlg::SendRechargeRequest @0x6d6d40 (renamed from sub_6D6D40;
// GMS_v79_1_DEVM.exe) builds COutPacket(59):
//
//	Encode1 op=2 (RECHARGE)  @0x6d6e63  (dispatcher prefix, not in body)
//	Encode2 slot             @0x6d6e6c
//
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v79 ida=0x6d6d40
func TestShopRechargeByteV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ShopRecharge{slot: 7}.Encode(l, ctx)(nil)
	want := []byte{0x07, 0x00} // slot=7  @0x6d6e6c
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 ShopRecharge: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v83 ida=0x756c28
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v87 ida=0x7a278f
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v95 ida=0x6e4e90
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=jms_v185 ida=0x7caecf
// packet-audit:verify packet=npc/serverbound/NpcShopRecharge version=gms_v84 ida=0x778edc
func TestShopRechargeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopRecharge{slot: 7}
			output := ShopRecharge{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
		})
	}
}
