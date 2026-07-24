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

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343, symbol-named) —
// CItemSpeakerDlg::_SendConsumeCashItemUseRequest@0x623728 (full decompile,
// own dedicated send function, NOT the main dispatcher — same architecture
// as gms_v95's CItemSpeakerDlg::_SendConsumeCashItemUseRequest@0x5c9e70):
//
//	COutPacket(&a3, 0x52)                                     // opcode 0x52 = USE_CASH_ITEM on v87
//	update_time = get_update_time(); Encode4(&a3, update_time) // header, FIRST
//	Encode2(&a3, *(this+144))                                  // slot
//	Encode4(&a3, *(this+148))                                  // itemId
//	EncodeStr(&a3, message)                                    // CCtrlEdit::GetText
//	Encode1(&a3, *(*(this+1504)+72))                           // whisper checkbox
//	Encode1(&a3, *(this+164) != 0)                             // hasItem
//	if (*(this+164)) {
//	  Encode4(&a3, *(this+152))                                // invType
//	  Encode4(&a3, *(this+156))                                // slot
//	}
//	SendPacket(...)
//
// update_time is written in THIS function's own header, before
// message/whisper/hasItem — confirming updateTimeFirst=TRUE for gms_v87 (same
// leading position as the main dispatcher's own header, and matching gms_v95).
//
// Wire (v87): message(str) + whisper(bool) + hasItem(bool) +
// [invType(int32) + slot(int32) iff hasItem]. Identical shape to gms_v95.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseItemMegaphone version=gms_v87 ida=0x623728
func TestItemUseItemMegaphoneByteOutputV87HasItem(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
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
		t.Errorf("v87 item use item megaphone (hasItem) golden mismatch: got %v want %v", actual, expected)
	}
}

func TestItemUseItemMegaphoneByteOutputV87NoItem(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
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
		t.Errorf("v87 item use item megaphone (no item) golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (jms_v185 MapleStory_dump_SCY.exe, port 13344, symbol-named) —
// CItemSpeakerDlg::_SendConsumeCashItemUseRequest@0x660672 (full decompile,
// own dedicated send function, NOT the main dispatcher — same architecture
// as gms_v87/v95's CItemSpeakerDlg::_SendConsumeCashItemUseRequest):
//
//	COutPacket::COutPacket(v22, 0x47)                          // opcode 0x47 = USE_CASH_ITEM on jms_v185
//	update_time = get_update_time(); Encode4(v22, update_time)  // header, FIRST
//	Encode2(v22, *(this+72))                                    // slot
//	Encode4(v22, *(this+37))                                    // itemId
//	EncodeStr(v22, CCtrlEdit::GetText(...))                     // message
//	Encode1(v22, *(*(this+617)+72))                             // whisper checkbox
//	Encode1(v22, *(this+41) != 0)                               // hasItem
//	if (*(this+41)) {
//	  Encode4(v22, *(this+38))                                  // invType
//	  Encode4(v22, *(this+39))                                  // slot
//	}
//	SendPacket(...)
//
// update_time is written in THIS function's own header, before
// message/whisper/hasItem — confirming updateTimeFirst=TRUE for jms_v185
// (matches the production gate: t.MajorVersion()>=87, jms185>=87, and the
// task-126 IDA citation for the main dispatcher @0xaef2f5 also showing
// leading update_time).
//
// Wire (jms_v185): message(str) + whisper(bool) + hasItem(bool) +
// [invType(int32) + slot(int32) iff hasItem]. Identical shape to gms_v87/v95.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseItemMegaphone version=jms_v185 ida=0x660672
func TestItemUseItemMegaphoneByteOutputJMS185HasItem(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
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
		t.Errorf("jms185 item use item megaphone (hasItem) golden mismatch: got %v want %v", actual, expected)
	}
}

func TestItemUseItemMegaphoneByteOutputJMS185NoItem(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
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
		t.Errorf("jms185 item use item megaphone (no item) golden mismatch: got %v want %v", actual, expected)
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
				// task-123 phase 3: matches the production gate exactly (see
				// item_use_megaphone_test.go for the IDA citation).
				updateTimeFirst := v.MajorVersion >= 87
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
