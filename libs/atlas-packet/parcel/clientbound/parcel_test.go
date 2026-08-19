package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestParcelEncode pins the PARCEL::Decode @0x4E4345 wire shape (v83): a
// fixed 234-byte block (id[4] + senderName[13] + mesos[4] + sentAt FILETIME[8]
// + message/padding[205]) then a hasItem bool and, if set, one
// GW_ItemSlotBase (model.Asset). The mode-prefix dispatcher these arms live
// under (CParcelDlg::OnPacket) decompiles byte-identically across all eight
// versions in docs/packets/dispatchers/parcel.yaml (Task 6 header
// derivation): gms_v72 @0x65f93a, gms_v79 @0x683709, gms_v83 @0x6f56ea,
// gms_v84 @0x70db9f, gms_v87 @0x7346db, gms_v92 @0x689190, gms_v95
// @0x692970, jms_v185 @0x755a21 (OPEN only).
//
// No `packet-audit:verify` markers yet: those require the per-version IDA
// export + evidence/audit-report pair (DISPATCHER_FAMILY.md step 5,
// VERIFYING_A_PACKET.md), which is a separate follow-up pass, not part of
// Task 7's struct/arm authoring scope.
func TestParcelEncode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("parcel no item", func(t *testing.T) {
		p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi")

		name := make([]byte, 13)
		copy(name, "Alice")
		msg := make([]byte, 205)
		copy(msg, "hi")
		filetime := model.MsTimeBytes(sentAt)

		var want []byte
		want = append(want, 0x07, 0x00, 0x00, 0x00) // id LE
		want = append(want, name...)
		want = append(want, 0xe8, 0x03, 0x00, 0x00) // mesos LE (1000)
		want = append(want, filetime[:]...)
		want = append(want, msg...)
		want = append(want, 0x00) // hasItem = false

		got := p.Encode(nil, ctx)(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("parcel no item: got %d bytes, want %d bytes\ngot:  % x\nwant: % x", len(got), len(want), got, want)
		}
		if len(got) != 234+1 {
			t.Errorf("parcel no item: total length got %d, want %d (234 fixed block + 1 hasItem byte)", len(got), 235)
		}
	})

	t.Run("parcel with item", func(t *testing.T) {
		item := model.NewAsset(false, 1, 1302000, time.Time{})
		p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi").SetItem(item)

		name := make([]byte, 13)
		copy(name, "Alice")
		msg := make([]byte, 205)
		copy(msg, "hi")
		filetime := model.MsTimeBytes(sentAt)
		itemBytes := item.Encode(nil, ctx)(nil)

		var want []byte
		want = append(want, 0x07, 0x00, 0x00, 0x00)
		want = append(want, name...)
		want = append(want, 0xe8, 0x03, 0x00, 0x00)
		want = append(want, filetime[:]...)
		want = append(want, msg...)
		want = append(want, 0x01) // hasItem = true
		want = append(want, itemBytes...)

		got := p.Encode(nil, ctx)(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("parcel with item: got % x\nwant % x", got, want)
		}
	})
}

// TestParcelOpenEncode pins the OPEN (mode 8, jms_v185 mode 10) and
// OPEN_QUICK (mode 0x1A, jms_v185 mode 0x1B) arm bodies.
// CTabReceive::SetParcel @0x6EF69C: bool quickEnabled, byte count +
// count*PARCEL (mailbox), byte newCount + newCount*PARCEL (arrived since
// last open). Mode bytes per docs/packets/dispatchers/parcel.yaml (Task 6).
//
// No `packet-audit:verify` markers yet — see TestParcelEncode's doc comment.
func TestParcelOpenEncode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi")
	pBytes := p.Encode(nil, ctx)(nil)

	t.Run("open empty mailbox", func(t *testing.T) {
		got := NewParcelOpen(8, true, nil, nil).Encode(nil, ctx)(nil)
		want := []byte{0x08, 0x01, 0x00, 0x00}
		if !bytes.Equal(got, want) {
			t.Errorf("open empty mailbox: got % x want % x", got, want)
		}
	})

	t.Run("open one parcel", func(t *testing.T) {
		got := NewParcelOpen(8, false, []parcel.Parcel{p}, nil).Encode(nil, ctx)(nil)
		var want []byte
		want = append(want, 0x08, 0x00, 0x01)
		want = append(want, pBytes...)
		want = append(want, 0x00)
		if !bytes.Equal(got, want) {
			t.Errorf("open one parcel: got % x want % x", got, want)
		}
	})

	t.Run("open with arrived", func(t *testing.T) {
		got := NewParcelOpen(8, false, []parcel.Parcel{p}, []parcel.Parcel{p}).Encode(nil, ctx)(nil)
		var want []byte
		want = append(want, 0x08, 0x00, 0x01)
		want = append(want, pBytes...)
		want = append(want, 0x01)
		want = append(want, pBytes...)
		if !bytes.Equal(got, want) {
			t.Errorf("open with arrived: got % x want % x", got, want)
		}
	})

	t.Run("open quick", func(t *testing.T) {
		got := NewParcelOpenQuick(0x1A).Encode(nil, ctx)(nil)
		want := []byte{0x1A}
		if !bytes.Equal(got, want) {
			t.Errorf("open quick: got % x want % x", got, want)
		}
	})
}
