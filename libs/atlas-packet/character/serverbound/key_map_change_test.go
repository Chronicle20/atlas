package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// KeyMapChange v48 byte-fixture — CHANGE_KEYMAP, op 110.
//
// Client send — CFuncKeyMappedMan::SaveFuncKeyMap @0x4e5fae: diffs the 89-entry
// keymap vs the saved copy, then COutPacket(110)@0x4e5fc8 + Encode4(0=mode)
// @0x4e5fd4 + Encode4(changedCount)@0x4e6021 + per-key{Encode4(keyIdx)@0x4e6035
// + EncodeBuffer(FUNCKEY_MAPPED, 5 bytes = nType[1]+nID[4]) via sub_49C937}.
// Per-entry layout = keyId(4) + theType(1) + action(4) = 9 bytes. No version
// gate; byte-identical to v61.
//
// packet-audit:verify packet=character/serverbound/KeyMapChange version=gms_v48 ida=0x4e5fae
func TestKeyMapChangeV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := KeyMapChange{
		mode: 0,
		entries: []KeyMapEntry{
			{KeyId: 2, TheType: 4, Action: 10},
			{KeyId: 16, TheType: 4, Action: 8},
		},
	}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0x00, 0x00, 0x00, 0x00, // mode = 0 (Encode4)      /*0x4e5fd4*/
		0x02, 0x00, 0x00, 0x00, // count = 2 (Encode4)     /*0x4e6021*/
		0x02, 0x00, 0x00, 0x00, // keyIdx 2 (Encode4)      /*0x4e6035*/
		0x04,                   // theType 4 (EncodeBuffer) /*0x49c937*/
		0x0a, 0x00, 0x00, 0x00, // action 10
		0x10, 0x00, 0x00, 0x00, // keyIdx 16 (Encode4)
		0x04,                   // theType 4
		0x08, 0x00, 0x00, 0x00, // action 8
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 KeyMapChange wire:\n got %x\nwant %x", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/KeyMapChange version=gms_v83 ida=0x58df2f
// packet-audit:verify packet=character/serverbound/KeyMapChange version=gms_v87 ida=0x5bd3f4
// packet-audit:verify packet=character/serverbound/KeyMapChange version=gms_v95 ida=0x568a60
// packet-audit:verify packet=character/serverbound/KeyMapChange version=jms_v185 ida=0x5e7b48
// packet-audit:verify packet=character/serverbound/KeyMapChange version=gms_v84 ida=0x59df22
func TestKeyMapChangeMode0RoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := KeyMapChange{
				mode: 0,
				entries: []KeyMapEntry{
					{KeyId: 2, TheType: 6, Action: 100},
					{KeyId: 63, TheType: 6, Action: 200},
				},
			}
			output := KeyMapChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if len(output.Entries()) != len(input.Entries()) {
				t.Fatalf("entries count: got %v, want %v", len(output.Entries()), len(input.Entries()))
			}
			for i, e := range output.Entries() {
				if e.KeyId != input.entries[i].KeyId {
					t.Errorf("entries[%d].KeyId: got %v, want %v", i, e.KeyId, input.entries[i].KeyId)
				}
				if e.TheType != input.entries[i].TheType {
					t.Errorf("entries[%d].TheType: got %v, want %v", i, e.TheType, input.entries[i].TheType)
				}
				if e.Action != input.entries[i].Action {
					t.Errorf("entries[%d].Action: got %v, want %v", i, e.Action, input.entries[i].Action)
				}
			}
		})
	}
}

func TestKeyMapChangeMode1RoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := KeyMapChange{mode: 1, itemId: 2001000}
			output := KeyMapChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}
