package serverbound

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationGift version=gms_v84 ida=0x472436
func TestShopOperationGiftRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationGift{birthday: 19900101, spw: "secret", serialNumber: 12345, oneADay: 1, name: "Player1", message: "Happy birthday!"}
			output := ShopOperationGift{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if v.Region == "JMS" {
				// JMS gift body carries only serialNumber; no birthday/spw/oneADay/name/message.
				if output.SerialNumber() != input.SerialNumber() {
					t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
				}
				if output.Birthday() != 0 {
					t.Errorf("birthday: got %v, want 0 for %s", output.Birthday(), v.Name)
				}
				if output.SPW() != "" {
					t.Errorf("spw: got %q, want empty for %s", output.SPW(), v.Name)
				}
				if output.OneADay() != 0 {
					t.Errorf("oneADay: got %v, want 0 for %s", output.OneADay(), v.Name)
				}
				if output.Name() != "" {
					t.Errorf("name: got %q, want empty for %s", output.Name(), v.Name)
				}
				if output.Message() != "" {
					t.Errorf("message: got %q, want empty for %s", output.Message(), v.Name)
				}
				return
			}
			// Leading field: spw string at v95+, birthday int at v87 and below.
			if v.Region == "GMS" && v.MajorVersion >= 95 {
				if output.SPW() != input.SPW() {
					t.Errorf("spw: got %v, want %v", output.SPW(), input.SPW())
				}
			} else if output.Birthday() != input.Birthday() {
				t.Errorf("birthday: got %v, want %v", output.Birthday(), input.Birthday())
			}
			// oneADay byte: present from v87 onward (v87 keeps the leading int).
			if v.Region == "GMS" && v.MajorVersion >= 87 {
				if output.OneADay() != input.OneADay() {
					t.Errorf("oneADay: got %v, want %v", output.OneADay(), input.OneADay())
				}
			} else if output.OneADay() != 0 {
				t.Errorf("oneADay should be absent (0) for %s, got %v", v.Name, output.OneADay())
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

// TestShopOperationGiftBytes pins the SPLIT version gate. IDA v83
// CCashShop::SendGiftsPacket@0x46f940: Encode4 A, Encode4 serialNumber, EncodeStr
// name, EncodeStr message (no oneADay). v87 SendGiftsPacket@0x47a168: STILL the
// leading Encode4 int (NOT SPW string), but inserts Encode1 oneADay before name.
// v95 @0x487b60: leading int replaced by EncodeStr sSPW, oneADay still present.
// Split gate: oneADay byte GMS && MajorVersion>=87; spw string GMS && MajorVersion>=95.
// packet-audit:verify packet=cash/serverbound/CashShopOperationGift version=gms_v83 ida=0x46f940
// packet-audit:verify packet=cash/serverbound/CashShopOperationGift version=gms_v87 ida=0x47a168
// packet-audit:verify packet=cash/serverbound/CashShopOperationGift version=gms_v95 ida=0x487b60
//
// v79 CCashShop::SendGiftsPacket@0x4691e3: COutPacket(221) Encode1(4)=mode (routed
// op), then Encode4(this[326]) leading int, Encode4(this[327])=serialNumber,
// EncodeStr(recipient name), EncodeStr(message). No oneADay byte (that is v87+),
// no SPW string (that is v95+). Body after the mode byte == the v83 shape.
// packet-audit:verify packet=cash/serverbound/CashShopOperationGift version=gms_v79 ida=0x4691e3
func TestShopOperationGiftBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationGift{birthday: 0x01020304, spw: "x", serialNumber: 0x05060708, oneADay: 1, name: "", message: ""}

	// v79: 04030201 | 08070605 | 0000 | 0000  (int + int + empty name + empty message, == v83)
	got79 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	if got79 != "04030201"+"08070605"+"0000"+"0000" {
		t.Errorf("v79 bytes: got %s", got79)
	}

	// v83: 04030201 | 08070605 | 0000 | 0000  (int + int + empty name + empty message)
	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	if got83 != "04030201"+"08070605"+"0000"+"0000" {
		t.Errorf("v83 bytes: got %s", got83)
	}

	// v87: 04030201 | 08070605 | 01 | 0000 | 0000  (leading int + serialNumber + oneADay byte + names)
	got87 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 87, 1))(nil))
	if got87 != "04030201"+"08070605"+"01"+"0000"+"0000" {
		t.Errorf("v87 bytes: got %s", got87)
	}

	// v95: 0100 78 | 08070605 | 01 | 0000 | 0000  (spw str + int + oneADay byte + names)
	got95 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil))
	if got95 != "010078"+"08070605"+"01"+"0000"+"0000" {
		t.Errorf("v95 bytes: got %s", got95)
	}
}

// TestShopOperationGiftV72Bytes pins the v72 legacy body. IDA v72
// CCashShop::SendGiftsPacket@0x467f9a (GMS_v72.1_U_DEVM.exe, port 13339):
// COutPacket(219) Encode1(4)=mode @0x4681ed (routed op) then Encode4(this[326])=
// birthday/leading @0x4681fb, Encode4(this[327])=serialNumber @0x468209,
// EncodeStr(name) @0x468222, EncodeStr(message) @0x46823e. No oneADay (v87+), no
// SPW (v95+). Body after the mode byte == v79.
// packet-audit:verify packet=cash/serverbound/CashShopOperationGift version=gms_v72 ida=0x467f9a
func TestShopOperationGiftV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationGift{birthday: 0x01020304, spw: "x", serialNumber: 0x05060708, oneADay: 1, name: "", message: ""}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "04030201"+"08070605"+"0000"+"0000" {
		t.Errorf("v72 bytes: got %s", got)
	}
}

// TestShopOperationGiftJMS pins the JMS185 gift body. IDA JMS185
// CCashShop::SendGiftsPacket@0x47bced: Encode1(0x2E) gift sub-op (routed as the
// op byte, NOT part of the body) then Encode4(commSN). The body after the sub-op
// is exactly serialNumber(4); no SPW/birthday, no recipient name, no message, no
// oneADay (NX-system divergence). Cross-checked against the JSON export read-order.
// packet-audit:verify packet=cash/serverbound/CashShopOperationGift version=jms_v185 ida=0x47bced
func TestShopOperationGiftJMS(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ShopOperationGift{birthday: 0x01020304, spw: "x", serialNumber: 0xAABBCCDD, oneADay: 1, name: "Player1", message: "Happy birthday!"}
	b := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
	// JMS body: serialNumber(4) = 4 bytes. No SPW/birthday/oneADay/name/message.
	if len(b) != 4 {
		t.Fatalf("JMS gift = %d bytes, want 4: % x", len(b), b)
	}
	if got := binary.LittleEndian.Uint32(b[0:4]); got != 0xAABBCCDD {
		t.Errorf("JMS serial = 0x%08x, want 0xAABBCCDD", got)
	}
}

// TestShopOperationGiftJMSRoundTrip confirms decodeJMS reads back what encodeJMS
// wrote (serialNumber only), leaving birthday/spw/oneADay/name/message at zero.
func TestShopOperationGiftJMSRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := ShopOperationGift{serialNumber: 0xAABBCCDD, name: "Player1", message: "hi"}
	output := ShopOperationGift{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.SerialNumber() != input.SerialNumber() {
		t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
	}
	if output.Birthday() != 0 {
		t.Errorf("birthday: got %v, want 0", output.Birthday())
	}
	if output.SPW() != "" {
		t.Errorf("spw: got %q, want empty", output.SPW())
	}
	if output.OneADay() != 0 {
		t.Errorf("oneADay: got %v, want 0", output.OneADay())
	}
	if output.Name() != "" {
		t.Errorf("name: got %q, want empty", output.Name())
	}
	if output.Message() != "" {
		t.Errorf("message: got %q, want empty", output.Message())
	}
}
