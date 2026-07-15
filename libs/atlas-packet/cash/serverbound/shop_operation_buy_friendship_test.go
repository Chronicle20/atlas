package serverbound

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyFriendship version=gms_v87 ida=0x47b293
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyFriendship version=gms_v84 ida=0x473550
func TestShopOperationBuyFriendshipRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationBuyFriendship{isPoints: true, currency: 2, flag: 1, birthday: 19900101, spw: "secret", option: 1, serialNumber: 12345, name: "Player1", message: "Friends forever"}
			output := ShopOperationBuyFriendship{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if v.Region == "GMS" && v.MajorVersion < 61 {
				// v48/v28 (GMS < 61): the friendship-ring buy carries a flag byte
				// in place of the currency int (isPoints + flag + serialNumber).
				if output.IsPoints() != input.IsPoints() {
					t.Errorf("isPoints: got %v, want %v", output.IsPoints(), input.IsPoints())
				}
				if output.Flag() != input.Flag() {
					t.Errorf("flag: got %v, want %v", output.Flag(), input.Flag())
				}
				if output.Currency() != 0 {
					t.Errorf("currency: got %v, want 0", output.Currency())
				}
				if output.SerialNumber() != input.SerialNumber() {
					t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
				}
				if output.Birthday() != 0 || output.Option() != 0 || output.Name() != "" || output.Message() != "" {
					t.Errorf("v48 should not carry birthday/option/name/message for %s", v.Name)
				}
				return
			}
			if v.Region == "GMS" && v.MajorVersion < 83 {
				// Legacy GMS (v79): isPoints + currency + serialNumber only.
				if output.IsPoints() != input.IsPoints() {
					t.Errorf("isPoints: got %v, want %v", output.IsPoints(), input.IsPoints())
				}
				if output.Currency() != input.Currency() {
					t.Errorf("currency: got %v, want %v", output.Currency(), input.Currency())
				}
				if output.SerialNumber() != input.SerialNumber() {
					t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
				}
				if output.Birthday() != 0 || output.Option() != 0 || output.Name() != "" || output.Message() != "" {
					t.Errorf("legacy should not carry birthday/option/name/message for %s", v.Name)
				}
				return
			}
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

// TestShopOperationBuyFriendshipLeadingFieldGate pins the leading-field gate. IDA
// v83 CCashShop::OnBuyFriendship@0x470a5a sends Encode4 ask_SPW (int); v95
// @0x491b30 sends EncodeStr sSPW. Gate GMS && MajorVersion>=95.
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyFriendship version=gms_v83 ida=0x470a5a
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyFriendship version=gms_v95 ida=0x491b30
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

// TestShopOperationBuyFriendshipJMS pins the JMS185 friendship body. IDA JMS185
// CCashShop::OnBuyFriendship@0x481184 (sub-op 0x24 consumed by routing):
// EncodeStr(SPW), Encode4(nCommSN), EncodeStr(recipient name), EncodeStr(message).
// JMS has NO option int and NO birthday; SPW leads (empty = 2-byte length 0).
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyFriendship version=jms_v185 ida=0x481184
func TestShopOperationBuyFriendshipJMS(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ShopOperationBuyFriendship{spw: "", serialNumber: 0xAABBCCDD, name: "Bob", message: "Hi"}
	b := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
	// Expected: spw(2)=0000 | serial(4) | name "Bob" (2+3) | message "Hi" (2+2)
	want := "0000" + "ddccbbaa" + "0300" + hex.EncodeToString([]byte("Bob")) + "0200" + hex.EncodeToString([]byte("Hi"))
	if got := hex.EncodeToString(b); got != want {
		t.Fatalf("JMS friendship body: got %s, want %s", got, want)
	}
	// Spot-check serial at offset 2 (after empty SPW length prefix).
	if got := binary.LittleEndian.Uint32(b[2:6]); got != 0xAABBCCDD {
		t.Errorf("JMS serial = 0x%08x, want 0xAABBCCDD", got)
	}
}

// TestShopOperationBuyFriendshipV79Bytes pins the v79 legacy body. IDA v79
// CCashShop::OnBuyFriendship@0x4671d5: COutPacket(221) Encode1(8)=mode (routed op)
// then Encode1(v24==2)=isPoints, Encode4(v24)=currency, Encode4(a2)=serialNumber.
// Body after the mode byte is isPoints(1)+currency(4)+serialNumber(4); no
// SPW/birthday/option and no recipient name/message.
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyFriendship version=gms_v79 ida=0x4671d5
func TestShopOperationBuyFriendshipV79Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyFriendship{isPoints: true, currency: 0x01020304, serialNumber: 0x05060708, birthday: 999, option: 9, name: "x", message: "y"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	if got != "01"+"04030201"+"08070605" {
		t.Errorf("v79 bytes: got %s, want 010403020108070605", got)
	}
}

// TestShopOperationBuyFriendshipV72Bytes pins the v72 legacy body. IDA v72
// CCashShop::OnBuyFriendship@0x46639e (GMS_v72.1_U_DEVM.exe, port 13339):
// COutPacket(219) Encode1(8)=mode @0x466510 (routed op) then Encode1(v24==2)=
// isPoints @0x466520, Encode4(v24)=currency @0x46652b, Encode4(a2)=serialNumber
// @0x466536. Body after the mode byte == v79: isPoints(1)+currency(4)+serial(4).
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyFriendship version=gms_v72 ida=0x46639e
func TestShopOperationBuyFriendshipV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyFriendship{isPoints: true, currency: 0x01020304, serialNumber: 0x05060708, birthday: 999, option: 9, name: "x", message: "y"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "01"+"04030201"+"08070605" {
		t.Errorf("v72 bytes: got %s, want 010403020108070605", got)
	}
}
