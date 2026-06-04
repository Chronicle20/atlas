package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestShopOperationRebateLockerItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationRebateLockerItem{birthday: 19900101, spw: "secret", unk: 123456789}
			output := ShopOperationRebateLockerItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if v.Region == "JMS" || (v.Region == "GMS" && v.MajorVersion >= 95) {
				if output.SPW() != input.SPW() {
					t.Errorf("spw: got %v, want %v", output.SPW(), input.SPW())
				}
			} else if output.Birthday() != input.Birthday() {
				t.Errorf("birthday: got %v, want %v", output.Birthday(), input.Birthday())
			}
			if output.Unk() != input.Unk() {
				t.Errorf("unk: got %v, want %v", output.Unk(), input.Unk())
			}
		})
	}
}

// TestShopOperationRebateLockerItemJMSBody pins the JMS body shape.
// JMS185 CCashShop::OnRebateLockerItem@0x47c059 (sub-op 0x1B consumed by
// routing): EncodeStr(SPW) then EncodeBuffer(8-byte locker SN). Empty SPW is a
// 2-byte zero-length prefix; the 8-byte buffer is WriteLong (uint64 LE).
func TestShopOperationRebateLockerItemJMSBody(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// spw empty -> 2-byte length prefix (0x0000); unk -> 8-byte LE identity.
	input := ShopOperationRebateLockerItem{birthday: 0xDEADBEEF, spw: "", unk: 0x0807060504030201}

	b := input.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)

	// empty SPW (2) + 8-byte buffer = 10 bytes
	if len(b) != 10 {
		t.Fatalf("JMS length: got %d, want 10", len(b))
	}
	if hex.EncodeToString(b[:2]) != "0000" {
		t.Errorf("JMS SPW prefix: got %s, want 0000", hex.EncodeToString(b[:2]))
	}
	// 8-byte identity, little-endian (WriteLong)
	if hex.EncodeToString(b[2:]) != "0102030405060708" {
		t.Errorf("JMS identity bytes: got %s, want 0102030405060708", hex.EncodeToString(b[2:]))
	}
}

// TestShopOperationRebateLockerItemLeadingFieldGate pins the leading-field gate.
// IDA v83 CCashShop::OnRebateLockerItem@0x46bde1 sends Encode4 ask_SPW (int) then
// EncodeBuffer 8; v95 @0x485840 sends EncodeStr sSPW then EncodeBuffer 8. Gate
// GMS && MajorVersion>=95.
func TestShopOperationRebateLockerItemLeadingFieldGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationRebateLockerItem{birthday: 0x01020304, spw: "x", unk: 0}

	// v83: leading int (4) + 8-byte buffer = 12 bytes
	b83 := input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if len(b83) != 12 {
		t.Errorf("v83 length: got %d, want 12", len(b83))
	}
	if hex.EncodeToString(b83)[:8] != "04030201" {
		t.Errorf("v83 leading: got %s", hex.EncodeToString(b83))
	}

	// v95: leading EncodeStr "x" (2+1=3) + 8-byte buffer = 11 bytes
	b95 := input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	if len(b95) != 11 {
		t.Errorf("v95 length: got %d, want 11", len(b95))
	}
	if hex.EncodeToString(b95)[:6] != "010078" {
		t.Errorf("v95 leading: got %s", hex.EncodeToString(b95))
	}
}
