package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// UPDATE_CHAR_BOX balloon, full-field case (roomType != 0). Verified
// byte-identical on gms_v83 (CUser::OnMiniRoomBalloon @ 0x938ba5) and gms_v95
// (@ 0x8e8d30) per ida-notes.md §G3; the leading characterId is consumed by
// the dispatcher CUserPool::OnUserCommonPacket (v83 @ 0x972401) before
// routing to the handler, not by OnMiniRoomBalloon itself.
// packet-audit:verify packet=interaction/clientbound/InteractionMiniRoomBalloon version=gms_v83 ida=0x938ba5
// packet-audit:verify packet=interaction/clientbound/InteractionMiniRoomBalloon version=gms_v95 ida=0x8e8d30
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
