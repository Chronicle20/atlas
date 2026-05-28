package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestShopOperationGiftRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationGift{birthday: 19900101, spw: "secret", serialNumber: 12345, oneADay: 1, name: "Player1", message: "Happy birthday!"}
			output := ShopOperationGift{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if v.Region == "GMS" && v.MajorVersion >= 95 {
				if output.SPW() != input.SPW() {
					t.Errorf("spw: got %v, want %v", output.SPW(), input.SPW())
				}
				if output.OneADay() != input.OneADay() {
					t.Errorf("oneADay: got %v, want %v", output.OneADay(), input.OneADay())
				}
			} else if output.Birthday() != input.Birthday() {
				t.Errorf("birthday: got %v, want %v", output.Birthday(), input.Birthday())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

// TestShopOperationGiftBytes pins the version gate. IDA v83
// CCashShop::SendGiftsPacket@0x46f940: Encode4 A, Encode4 serialNumber, EncodeStr
// name, EncodeStr message (no oneADay). v95 @0x487b60: EncodeStr sSPW, Encode4
// serialNumber, Encode1 oneADay, EncodeStr name, EncodeStr message. Gate GMS &&
// MajorVersion>=95.
func TestShopOperationGiftBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationGift{birthday: 0x01020304, spw: "x", serialNumber: 0x05060708, oneADay: 1, name: "", message: ""}

	// v83: 04030201 | 08070605 | 0000 | 0000  (int + int + empty name + empty message)
	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	if got83 != "04030201"+"08070605"+"0000"+"0000" {
		t.Errorf("v83 bytes: got %s", got83)
	}

	// v95: 0100 78 | 08070605 | 01 | 0000 | 0000  (spw str + int + oneADay byte + names)
	got95 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil))
	if got95 != "010078"+"08070605"+"01"+"0000"+"0000" {
		t.Errorf("v95 bytes: got %s", got95)
	}
}
