package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// The GMS v83 case-17 arm of CWvsContext::SendConsumeCashItemUseRequest
// (fn entry @0xa0a63f; jumptable arm entry @0xa0ba15) performs exactly ONE
// encode — COutPacket::EncodeStr @0xa0bcb5 (re-derived directly against the
// v83 IDB, MapleStory_dump.exe.i64, task-224 fix round 1) — and then falls
// through (jmp loc_A0E9EC @0xa0bce5) to the shared dispatcher tail, which
// appends update_time on the builds that trail it. The `ida=` marker below
// cites the FUNCTION ENTRY address (0xa0a63f), matching every sibling
// candidate of this shared multi-arm sender (see the CashItemUseMegaphone /
// CashItemUseTeleportRock / etc. markers in this package) — evidence and the
// audit report are pinned at the function level, not the specific call site,
// since candidatesFromFName resolves ALL of this sender's sub-arms to the
// same export entry.
// packet-audit:verify packet=cash/serverbound/CashItemUsePetNameTag version=gms_v83 ida=0xa0a63f
func TestItemUsePetNameTagBytesTrailingUpdateTime(t *testing.T) {
	m := ItemUsePetNameTag{name: "Fluffy", updateTime: 0x01020304, updateTimeFirst: false}
	got := m.Encode(nil, nil)(nil)
	want := []byte{
		0x06, 0x00, // name length
		'F', 'l', 'u', 'f', 'f', 'y',
		0x04, 0x03, 0x02, 0x01, // trailing update_time
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

// GMS v87+ and JMS v185 lead update_time in the common ItemUse header, so the
// sub-body is the string alone (cash/serverbound/item_use.go UpdateTimeFirst).
func TestItemUsePetNameTagBytesLeadingUpdateTime(t *testing.T) {
	m := ItemUsePetNameTag{name: "Fluffy", updateTime: 0x01020304, updateTimeFirst: true}
	got := m.Encode(nil, nil)(nil)
	want := []byte{
		0x06, 0x00,
		'F', 'l', 'u', 'f', 'f', 'y',
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

func TestItemUsePetNameTagDecodeRoundTrip(t *testing.T) {
	for _, first := range []bool{false, true} {
		src := ItemUsePetNameTag{name: "Rex", updateTime: 0x0A0B0C0D, updateTimeFirst: first}
		b := src.Encode(nil, nil)(nil)

		req := request.Request(b)
		reader := request.NewRequestReader(&req, 0)

		dst := NewItemUsePetNameTag(first)
		dst.Decode(nil, nil)(&reader, nil)

		if dst.Name() != "Rex" {
			t.Fatalf("first=%t name = %q, want %q", first, dst.Name(), "Rex")
		}
		if !first && dst.UpdateTime() != 0x0A0B0C0D {
			t.Fatalf("updateTime = %X, want 0A0B0C0D", dst.UpdateTime())
		}
	}
}
