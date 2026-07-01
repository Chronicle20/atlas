package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationGetPurchaseRecord version=gms_v95 ida=0x4823c0
// packet-audit:verify packet=cash/serverbound/CashShopOperationGetPurchaseRecord version=gms_v87 ida=0x475b91
// packet-audit:verify packet=cash/serverbound/CashShopOperationGetPurchaseRecord version=gms_v83 ida=0x46bd0e
// packet-audit:verify packet=cash/serverbound/CashShopOperationGetPurchaseRecord version=jms_v185 ida=0x47bf86
// packet-audit:verify packet=cash/serverbound/CashShopOperationGetPurchaseRecord version=gms_v84 ida=0x46e300
//
// v79 (CCashShop::RequestCashPurchaseRecord @0x4667eb): COutPacket(221=0xDD
// CASHSHOP_OPERATION sb op) + Encode1(0x28)=sub-op mode + Encode4(serialNumber).
// The op byte and mode byte are handled by the dispatch; body = Encode4 = this
// codec's WriteInt(serialNumber). No MajorVersion gate (== v83).
// packet-audit:verify packet=cash/serverbound/CashShopOperationGetPurchaseRecord version=gms_v79 ida=0x4667eb
func TestShopOperationGetPurchaseRecordRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationGetPurchaseRecord{serialNumber: 12345}
			output := ShopOperationGetPurchaseRecord{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}

// TestShopOperationGetPurchaseRecordV72Bytes pins the v72 body. IDA v72
// CCashShop::RequestCashPurchaseRecord@0x4659b4 (GMS_v72.1_U_DEVM.exe, port
// 13339): COutPacket(219) Encode1(0x27)=mode @0x4659d7 (routed op, dispatcher
// -supplied) then Encode4(a2)=serialNumber @0x4659e2. Body after the mode byte is
// exactly serialNumber(4), == every version (no MajorVersion gate).
// packet-audit:verify packet=cash/serverbound/CashShopOperationGetPurchaseRecord version=gms_v72 ida=0x4659b4
func TestShopOperationGetPurchaseRecordV72Bytes(t *testing.T) {
	input := ShopOperationGetPurchaseRecord{serialNumber: 0x05060708}
	got := input.Encode(nil, pt.CreateContext("GMS", 72, 1))(nil)
	want := []byte{0x08, 0x07, 0x06, 0x05} // serialNumber uint32 LE — Encode4 @0x4659e2
	if !bytes.Equal(got, want) {
		t.Errorf("v72 bytes: got % x, want % x", got, want)
	}
}
