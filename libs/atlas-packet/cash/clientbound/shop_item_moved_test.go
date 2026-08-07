package clientbound

import (
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CashItemMovedToCashInventory corresponds to OnCashItemResMoveStoLDone@0x4948d0
// (case 0x79 MOVE_S_TO_L_DONE = item moved storage->locker = into the cash
// inventory). Body after the mode byte: DecodeBuffer(v6, 0x37u) = 55-byte
// GW_CashItemInfo (CInPacket::DecodeBuffer @0x4948d0). The wire is mode +
// CashInventoryItem.EncodeBytes (55 bytes).
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToCashInventory version=gms_v95 ida=0x4948d0
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToCashInventory version=gms_v83 ida=0x47b2fd
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToCashInventory version=gms_v84 ida=0x47e49b
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToCashInventory version=gms_v87 ida=0x486ad3
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToCashInventory version=jms_v185 ida=0x48ded7
func TestCashItemMovedToCashInventoryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCashItemMovedToCashInventory(0x50, testItem())
			output := CashItemMovedToCashInventory{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Item().CashId != input.Item().CashId {
				t.Errorf("cashId: got %v, want %v", output.Item().CashId, input.Item().CashId)
			}
			if output.Item().TemplateId != input.Item().TemplateId {
				t.Errorf("templateId: got %v, want %v", output.Item().TemplateId, input.Item().TemplateId)
			}
		})
	}
}

// CashItemMovedToInventory corresponds to OnCashItemResMoveLtoSDone@0x495050
// (case 0x77 MOVE_L_TO_S_DONE = item moved locker->slot = into the player
// inventory). Body after the mode byte: nPOS = Decode2(iPacket) then
// GW_ItemSlotBase::Decode(&pItem, iPacket). The leading wire is mode (1 byte)
// + slot (Decode2 = 2-byte LE short); the trailing item-slot payload is the
// opaque model.Asset (register boundary — verified via model.Asset's own
// encoder, not byte-cited here).
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v95 ida=0x495050
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v83 ida=0x47aee2
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v84 ida=0x47e080
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=gms_v87 ida=0x4866b4
// packet-audit:verify packet=cash/clientbound/CashCashItemMovedToInventory version=jms_v185 ida=0x48dab8
func TestCashItemMovedToInventoryBytePrefix(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// non-zero asset so the opaque payload is present; only the leading
	// mode + slot prefix is byte-asserted (the asset is the opaque boundary).
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(5, 0, 0)
	input := NewCashItemMovedToInventory(0x77, 0x0102, asset)
	got := input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	// byte 0: mode 0x77 (Decode1; case 0x77 consumed by dispatcher)
	if got[0] != 0x77 {
		t.Errorf("mode byte: got 0x%02X, want 0x77", got[0])
	}
	// bytes 1-2: slot 0x0102 little-endian (Decode2 nPOS) -> 0x02 0x01
	if got[1] != 0x02 || got[2] != 0x01 {
		t.Errorf("slot bytes: got 0x%02X 0x%02X, want 0x02 0x01", got[1], got[2])
	}
}

func TestCashItemMovedToInventoryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			asset := model.NewAsset(true, 0, 2000000, time.Time{}).
				SetStackableInfo(5, 0, 0)
			input := NewCashItemMovedToInventory(0x51, 3, asset)
			output := CashItemMovedToInventory{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Asset().TemplateId() != input.Asset().TemplateId() {
				t.Errorf("templateId: got %v, want %v", output.Asset().TemplateId(), input.Asset().TemplateId())
			}
		})
	}
}
