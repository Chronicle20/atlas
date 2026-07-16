package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestMiniRoomBalloonBytes asserts the EXACT UPDATE_CHAR_BOX balloon wire
// sequence (full-field case, roomType != 0) against the documented
// CUser::OnMiniRoomBalloon read order (ida-notes.md §G3), so a symmetric
// field-order bug in the encoder+test-decoder pair (which a RoundTrip-only
// test would miss) is caught. The encoder is version-uniform, so one GMS
// context suffices.
func TestMiniRoomBalloonBytes(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	input := NewMiniRoomBalloon(1234, 1, 1234, "Omok Room", true, 1, 1, 2, false)

	got := test.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, le32(1234)...)         // characterId
	want = append(want, 1)                     // roomType
	want = append(want, le32(1234)...)         // roomId
	want = append(want, ascii("Omok Room")...) // title
	want = append(want, 1)                     // hasPassword (true)
	want = append(want, 1)                     // pieceType
	want = append(want, 1)                     // occupancy
	want = append(want, 2)                     // capacity
	want = append(want, 0)                     // inProgress (false)

	if !bytes.Equal(got, want) {
		t.Fatalf("byte mismatch:\n got  %x\n want %x", got, want)
	}
}

// UPDATE_CHAR_BOX balloon, full-field case (roomType != 0). Verified
// byte-identical on gms_v83 (CUser::OnMiniRoomBalloon @ 0x938ba5) and gms_v95
// (@ 0x8e8d30) per ida-notes.md §G3; the leading characterId is consumed by
// the dispatcher CUserPool::OnUserCommonPacket (v83 @ 0x972401) before
// routing to the handler, not by OnMiniRoomBalloon itself.
// packet-audit:verify packet=interaction/clientbound/InteractionMiniRoomBalloon version=gms_v83 ida=0x938ba5
// packet-audit:verify packet=interaction/clientbound/InteractionMiniRoomBalloon version=gms_v95 ida=0x8e8d30
// Legacy (task-133 do-mode): CUser::OnMiniRoomBalloon reads the identical order
// on gms_v61 (@0x7920b9), gms_v72 (@0x847df1) and gms_v79 (@0x8922ce) —
// Decode1(roomType; ==0 => destroy, no trailing) / Decode4(roomId) /
// DecodeStr(title) / 5x Decode1 (hasPassword, pieceType, occupancy, capacity,
// inProgress). The leading characterId is consumed upstream by the user-packet
// dispatcher, not by OnMiniRoomBalloon. Byte-identical to v83/v95 — no version
// gate needed.
// packet-audit:verify packet=interaction/clientbound/InteractionMiniRoomBalloon version=gms_v61 ida=0x7920b9
// packet-audit:verify packet=interaction/clientbound/InteractionMiniRoomBalloon version=gms_v72 ida=0x847df1
// packet-audit:verify packet=interaction/clientbound/InteractionMiniRoomBalloon version=gms_v79 ida=0x8922ce
func TestMiniRoomBalloonRoundTrip(t *testing.T) {
	input := NewMiniRoomBalloon(1234, 1, 1234, "Omok Room", true, 1, 1, 2, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&MiniRoomBalloon{}).Decode, nil)
		})
	}
}

// UPDATE_CHAR_BOX balloon removal (roomType == 0): the trailing fields are
// absent from the wire per ida-notes.md §G3 ("byte roomType # 0 = remove
// balloon"; no trailing fields when roomType is 0). Same handler
// (CUser::OnMiniRoomBalloon) as MiniRoomBalloon above.
// packet-audit:verify packet=interaction/clientbound/InteractionMiniRoomBalloonRemove version=gms_v83 ida=0x938ba5
// packet-audit:verify packet=interaction/clientbound/InteractionMiniRoomBalloonRemove version=gms_v95 ida=0x8e8d30
func TestMiniRoomBalloonRemoveRoundTrip(t *testing.T) {
	input := NewMiniRoomBalloonRemove(1234)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, (&MiniRoomBalloonRemove{}).Decode, nil)
		})
	}
}
