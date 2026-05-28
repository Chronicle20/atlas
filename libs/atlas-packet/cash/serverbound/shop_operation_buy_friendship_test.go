package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestShopOperationBuyFriendshipRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationBuyFriendship{birthday: 19900101, spw: "secret", option: 1, serialNumber: 12345, name: "Player1", message: "Friends forever"}
			output := ShopOperationBuyFriendship{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if v.Region == "GMS" && v.MajorVersion >= 95 {
				if output.SPW() != input.SPW() {
					t.Errorf("spw: got %v, want %v", output.SPW(), input.SPW())
				}
			} else if output.Birthday() != input.Birthday() {
				t.Errorf("birthday: got %v, want %v", output.Birthday(), input.Birthday())
			}
			if output.Option() != input.Option() {
				t.Errorf("option: got %v, want %v", output.Option(), input.Option())
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

// TestShopOperationBuyFriendshipLeadingFieldGate pins the leading-field gate. IDA
// v83 CCashShop::OnBuyFriendship@0x470a5a sends Encode4 ask_SPW (int); v95
// @0x491b30 sends EncodeStr sSPW. Gate GMS && MajorVersion>=95.
func TestShopOperationBuyFriendshipLeadingFieldGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyFriendship{birthday: 0x01020304, spw: "x", option: 0, serialNumber: 0, name: "", message: ""}

	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	if got83[:8] != "04030201" {
		t.Errorf("v83 leading: got %s, want 04030201...", got83)
	}

	got95 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil))
	if got95[:6] != "010078" {
		t.Errorf("v95 leading: got %s, want 010078...", got95)
	}
}
