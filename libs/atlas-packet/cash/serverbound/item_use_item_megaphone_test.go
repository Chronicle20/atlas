package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CItemSpeakerDlg::_SendConsumeCashItemUseRequest@0x5c9e70 (the item
// megaphone's own OK-button handler; the main SendConsumeCashItemUseRequest
// dispatcher's jumptable case 14 @0x9ebca1 only constructs and shows the
// CItemSpeakerDlg — this SEPARATE small function is what actually sends,
// exactly matching the task-19 brief's hint that this class carries the
// real send function). Decompile:
//
//	COutPacket::COutPacket(&oPacket, 85)                    // opcode 0x55
//	update_time = get_update_time(); Encode4(update_time)   // header, FIRST
//	Encode2(this->_nPOS)
//	Encode4(this->_nItemID)
//	EncodeStr(CCtrlEdit::GetText(_pEditInput))               // message
//	Encode1(this->_pCheckBoxWhisper.p->m_bChecked)            // whisper
//	Encode1(this->_pItem.p != 0)                              // hasItem
//	if (this->_pItem.p) {
//	  Encode4(this->_nTargetTI)                               // invType
//	  Encode4(this->_nTargetPOS)                               // slot
//	}
//	SendPacket(...)
//
// This DEFINITIVELY cracks the cell the gms_v83 pass left BLOCKED (field
// layout was not pinned down there). update_time is written in THIS
// function's own header, before message/whisper/hasItem — confirming
// updateTimeFirst=TRUE for gms_v95 in this independent send path too (same
// conclusion as the shared SendConsumeCashItemUseRequest header). No
// trailing updateTime after hasItem/invType/slot.
//
// Wire (v95): message(str) + whisper(bool) + hasItem(bool) +
// [invType(int32) + slot(int32) iff hasItem]. Matches
// ItemUseItemMegaphone.Encode(updateTimeFirst=true) exactly (invType<-nTargetTI,
// slot<-nTargetPOS).
//
// packet-audit:verify packet=cash/serverbound/CashItemUseItemMegaphone version=gms_v95 ida=0x5c9e70
func TestItemUseItemMegaphoneByteOutputV95HasItem(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItemUseItemMegaphone(true)
	input.message = "Item hello!"
	input.whisper = true
	input.hasItem = true
	input.invType = 2
	input.slot = 5
	expected := []byte{
		0x0B, 0x00, 'I', 't', 'e', 'm', ' ', 'h', 'e', 'l', 'l', 'o', '!', // message
		0x01,                   // whisper=true
		0x01,                   // hasItem=true
		0x02, 0x00, 0x00, 0x00, // invType=2 LE
		0x05, 0x00, 0x00, 0x00, // slot=5 LE
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 item use item megaphone (hasItem) golden mismatch: got %v want %v", actual, expected)
	}
}

func TestItemUseItemMegaphoneByteOutputV95NoItem(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItemUseItemMegaphone(true)
	input.message = "no item"
	input.whisper = false
	input.hasItem = false
	expected := []byte{
		0x07, 0x00, 'n', 'o', ' ', 'i', 't', 'e', 'm', // message
		0x00, // whisper=false
		0x00, // hasItem=false
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 item use item megaphone (no item) golden mismatch: got %v want %v", actual, expected)
	}
}

func TestItemUseItemMegaphoneRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		whisper bool
		hasItem bool
		invType int32
		slot    int32
	}{
		{"hasItem_true", false, true, 2, 5},
		{"hasItem_false", true, false, 0, 0},
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
				input := NewItemUseItemMegaphone(updateTimeFirst)
				input.message = "Item hello!"
				input.whisper = tc.whisper
				input.hasItem = tc.hasItem
				input.invType = tc.invType
				input.slot = tc.slot
				if !updateTimeFirst {
					input.updateTime = 99999
				}
				output := NewItemUseItemMegaphone(updateTimeFirst)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Message() != input.Message() {
					t.Errorf("message: got %q, want %q", output.Message(), input.Message())
				}
				if output.Whisper() != input.Whisper() {
					t.Errorf("whisper: got %v, want %v", output.Whisper(), input.Whisper())
				}
				if output.HasItem() != input.HasItem() {
					t.Errorf("hasItem: got %v, want %v", output.HasItem(), input.HasItem())
				}
				if tc.hasItem {
					if output.InvType() != input.InvType() {
						t.Errorf("invType: got %v, want %v", output.InvType(), input.InvType())
					}
					if output.Slot() != input.Slot() {
						t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
					}
				}
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}
