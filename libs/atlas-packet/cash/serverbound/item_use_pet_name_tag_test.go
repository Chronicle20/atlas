package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// The GMS v83 case-17 arm of CWvsContext::SendConsumeCashItemUseRequest
// (arm entry @0xa0ba15) performs exactly ONE encode — COutPacket::EncodeStr
// @0xa0bcb5 — and then falls through to the shared tail at loc_A0E9EC, which
// appends update_time on the builds that trail it.
// packet-audit:verify packet=cash/serverbound/ItemUsePetNameTag version=gms_v83 ida=0xa0bcb5
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
