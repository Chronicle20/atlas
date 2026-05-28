package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

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
			if output.Currency() != input.Currency() {
				t.Errorf("currency: got %v, want %v", output.Currency(), input.Currency())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if v.Region == "GMS" && v.MajorVersion >= 95 {
				if output.OneADay() != input.OneADay() {
					t.Errorf("oneADay: got %v, want %v", output.OneADay(), input.OneADay())
				}
				if output.EventSN() != input.EventSN() {
					t.Errorf("eventSN: got %v, want %v", output.EventSN(), input.EventSN())
				}
			} else if output.Zero() != input.Zero() {
				t.Errorf("zero: got %v, want %v", output.Zero(), input.Zero())
			}
		})
	}
}

// TestShopOperationBuyBytes pins the version gate on the tail. IDA v83
// CCashShop::OnBuy@0x46dadd: Encode1 isMaplePoint, Encode4 dwOption, Encode4
// nCommSN, Encode4 IsZeroGoods (single int). v95 @0x48e530: ...nCommSN then
// Encode1 m_bRequestBuyOneADay + Encode4 nEventSN. Gate GMS && MajorVersion>=95.
func TestShopOperationBuyBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuy{isPoints: true, currency: 1, serialNumber: 2, zero: 3, oneADay: 1, eventSN: 4}

	// v83: 01 | 01000000 | 02000000 | 03000000  (4-byte zero tail)
	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	if got83 != "01"+"01000000"+"02000000"+"03000000" {
		t.Errorf("v83 bytes: got %s", got83)
	}

	// v95: 01 | 01000000 | 02000000 | 01 | 04000000  (byte oneADay + int eventSN)
	got95 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil))
	if got95 != "01"+"01000000"+"02000000"+"01"+"04000000" {
		t.Errorf("v95 bytes: got %s", got95)
	}
}
