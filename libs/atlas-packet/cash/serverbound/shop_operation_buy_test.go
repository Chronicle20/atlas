package serverbound

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationBuy version=gms_v95 ida=0x48e530
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuy version=gms_v84 ida=0x47036b
func TestShopOperationBuyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationBuy{isPoints: true, currency: 1, serialNumber: 12345, zero: 7, oneADay: 1, eventSN: 99}
			output := ShopOperationBuy{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IsPoints() != input.IsPoints() {
				t.Errorf("isPoints: got %v, want %v", output.IsPoints(), input.IsPoints())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if v.Region == "JMS" {
				// JMS body carries only isPoints + serialNumber; no currency/tail.
				if output.Currency() != 0 {
					t.Errorf("currency: got %v, want 0", output.Currency())
				}
				return
			}
			if buyOmitsCurrency(tenant.MustFromContext(ctx)) {
				// GMS < 61 (v48/v28) sends only isPoints+serialNumber; the
				// currency int is version-absent (added at v61).
				if output.Currency() != 0 {
					t.Errorf("currency: got %v, want 0", output.Currency())
				}
				return
			}
			if output.Currency() != input.Currency() {
				t.Errorf("currency: got %v, want %v", output.Currency(), input.Currency())
			}
			if v.Region == "GMS" && v.MajorVersion >= 87 {
				if output.OneADay() != input.OneADay() {
					t.Errorf("oneADay: got %v, want %v", output.OneADay(), input.OneADay())
				}
				if output.EventSN() != input.EventSN() {
					t.Errorf("eventSN: got %v, want %v", output.EventSN(), input.EventSN())
				}
			} else if !buyOmitsTrailingZero(tenant.MustFromContext(ctx)) && output.Zero() != input.Zero() {
				// GMS < 72 (v61) omits the trailing IsZeroGoods int entirely.
				t.Errorf("zero: got %v, want %v", output.Zero(), input.Zero())
			}
		})
	}
}

// TestShopOperationBuyBytes pins the version gate on the tail. IDA v83
// CCashShop::OnBuy@0x46dadd: Encode1 isMaplePoint, Encode4 dwOption, Encode4
// nCommSN, Encode4 IsZeroGoods (single int). v87 CCashShop::OnBuy@0x477bd9
// already sends ...nCommSN then Encode1 m_bRequestBuyOneADay + Encode4 nEventSN
// (the byte+eventSN tail is present from v87, not v95). Gate GMS && MajorVersion>=87.
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuy version=gms_v83 ida=0x46dadd
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuy version=gms_v87 ida=0x477bd9
//
// v79 CCashShop::OnBuy@0x467f58: COutPacket(221) Encode1(3)=mode (routed op),
// then Encode1(v38==2)=isPoints, Encode4(v38)=currency, Encode4(a2)=serialNumber,
// Encode4(v34)=trailing zero/bundle int. Body after the mode byte is exactly the
// v83 shape (bool + int + int + 4-byte tail); no v87 oneADay/eventSN. v79<87 gate
// takes the else branch, identical to v83.
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuy version=gms_v79 ida=0x467f58
func TestShopOperationBuyBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuy{isPoints: true, currency: 1, serialNumber: 2, zero: 3, oneADay: 1, eventSN: 4}

	// v79: 01 | 01000000 | 02000000 | 03000000  (4-byte zero tail, == v83)
	got79 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	if got79 != "01"+"01000000"+"02000000"+"03000000" {
		t.Errorf("v79 bytes: got %s", got79)
	}

	// v83: 01 | 01000000 | 02000000 | 03000000  (4-byte zero tail)
	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	if got83 != "01"+"01000000"+"02000000"+"03000000" {
		t.Errorf("v83 bytes: got %s", got83)
	}

	// v87: 01 | 01000000 | 02000000 | 01 | 04000000  (byte oneADay + int eventSN, matches v95)
	got87 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 87, 1))(nil))
	if got87 != "01"+"01000000"+"02000000"+"01"+"04000000" {
		t.Errorf("v87 bytes: got %s", got87)
	}

	// v95: 01 | 01000000 | 02000000 | 01 | 04000000  (byte oneADay + int eventSN)
	got95 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil))
	if got95 != "01"+"01000000"+"02000000"+"01"+"04000000" {
		t.Errorf("v95 bytes: got %s", got95)
	}
}

// TestShopOperationBuyV72Bytes pins the v72 body. IDA v72 CCashShop::OnBuy
// @0x466e70 (GMS_v72.1_U_DEVM.exe, port 13339): COutPacket(219) Encode1(3)=mode
// @0x467355 (routed op), then Encode1(v47==2)=isPoints @0x467365, Encode4(v47)=
// currency @0x467370, Encode4(a2)=serialNumber @0x46737c, Encode4(v43)=trailing
// @0x467387. Body after the mode byte == v79 (v72<87 takes the legacy else branch).
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuy version=gms_v72 ida=0x466e70
func TestShopOperationBuyV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuy{isPoints: true, currency: 1, serialNumber: 2, zero: 3, oneADay: 1, eventSN: 4}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "01"+"01000000"+"02000000"+"03000000" {
		t.Errorf("v72 bytes: got %s", got)
	}
}

// TestShopOperationBuyJMS pins the JMS185 buy body. IDA JMS185
// CCashShop::OnBuy@0x47eaa7 normal-buy SendPacket: Encode1(3) mode (routed as
// op byte, not body), Encode1(usePoints), Encode4(nCommSN). The body after the
// mode byte is exactly isPoints(1) + serialNumber(4) = 5 bytes; no currency, no
// trailing v83/v87 fields.
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuy version=jms_v185 ida=0x47eaa7
func TestShopOperationBuyJMS(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ShopOperationBuy{isPoints: true, serialNumber: 0xAABBCCDD}
	b := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
	// JMS body: isPoints(1) + serialNumber(4) = 5 bytes. No currency, no trailing.
	if len(b) != 5 {
		t.Fatalf("JMS buy = %d bytes, want 5: % x", len(b), b)
	}
	if b[0] != 0x01 {
		t.Errorf("JMS isPoints byte = 0x%02x, want 0x01", b[0])
	}
	if got := binary.LittleEndian.Uint32(b[1:5]); got != 0xAABBCCDD {
		t.Errorf("JMS serial = 0x%08x, want 0xAABBCCDD", got)
	}
}

// TestShopOperationBuyJMSRoundTrip confirms decodeJMS reads back what encodeJMS
// wrote (isPoints + serialNumber), leaving currency/zero/oneADay/eventSN at zero.
func TestShopOperationBuyJMSRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := ShopOperationBuy{isPoints: true, serialNumber: 0xAABBCCDD}
	output := ShopOperationBuy{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.IsPoints() != input.IsPoints() {
		t.Errorf("isPoints: got %v, want %v", output.IsPoints(), input.IsPoints())
	}
	if output.SerialNumber() != input.SerialNumber() {
		t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
	}
	if output.Currency() != 0 {
		t.Errorf("currency: got %v, want 0", output.Currency())
	}
	if output.Zero() != 0 {
		t.Errorf("zero: got %v, want 0", output.Zero())
	}
	if output.OneADay() != 0 {
		t.Errorf("oneADay: got %v, want 0", output.OneADay())
	}
	if output.EventSN() != 0 {
		t.Errorf("eventSN: got %v, want 0", output.EventSN())
	}
}
