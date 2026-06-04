package serverbound

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestShopOperationBuyCoupleRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationBuyCouple{birthday: 19900101, spw: "secret", option: 1, serialNumber: 12345, name: "Player1", message: "Hello"}
			output := ShopOperationBuyCouple{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if v.Region == "JMS" {
				// JMS body: spw + serialNumber + name + message; no birthday, no option.
				if output.SPW() != input.SPW() {
					t.Errorf("spw: got %v, want %v", output.SPW(), input.SPW())
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
				if output.Option() != 0 {
					t.Errorf("option: got %v, want 0", output.Option())
				}
				return
			}
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

// TestShopOperationBuyCoupleLeadingFieldGate pins the leading-field gate. IDA v83
// CCashShop::OnBuyCouple@0x46ffe7 sends Encode4 ask_SPW (int); v95 @0x490d80 sends
// EncodeStr sSPW. Gate GMS && MajorVersion>=95.
func TestShopOperationBuyCoupleLeadingFieldGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyCouple{birthday: 0x01020304, spw: "x", option: 0, serialNumber: 0, name: "", message: ""}

	// v83 leading int 0x01020304 (LE)
	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	if got83[:8] != "04030201" {
		t.Errorf("v83 leading: got %s, want 04030201...", got83)
	}

	// v95 leading EncodeStr "x" = 0100 78
	got95 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil))
	if got95[:6] != "010078" {
		t.Errorf("v95 leading: got %s, want 010078...", got95)
	}
}

// TestShopOperationBuyCoupleJMS pins the JMS185 couple body. IDA JMS185
// CCashShop::OnBuyCouple@0x48085a (sub-op 0x1E consumed by routing): EncodeStr(SPW),
// Encode4(nCommSN), EncodeStr(sGiveTo recipient), EncodeStr(sText message). JMS has
// NO option int and NO birthday; SPW leads (empty = 1-byte length 0).
func TestShopOperationBuyCoupleJMS(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ShopOperationBuyCouple{spw: "", serialNumber: 0xAABBCCDD, name: "Bob", message: "Hi"}
	b := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
	// Expected: spw(2)=0000 | serial(4) | name "Bob" (2+3) | message "Hi" (2+2)
	want := "0000" + "ddccbbaa" + "0300" + hex.EncodeToString([]byte("Bob")) + "0200" + hex.EncodeToString([]byte("Hi"))
	if got := hex.EncodeToString(b); got != want {
		t.Fatalf("JMS couple body: got %s, want %s", got, want)
	}
	// Spot-check serial at offset 2 (after empty SPW length prefix).
	if got := binary.LittleEndian.Uint32(b[2:6]); got != 0xAABBCCDD {
		t.Errorf("JMS serial = 0x%08x, want 0xAABBCCDD", got)
	}
}
