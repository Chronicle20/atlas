package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// updateByteAvatar returns an avatar with a single equipment slot (and no
// masked/pet entries) so the model.Avatar map iteration is deterministic and
// the trailing opaque blob is byte-stable across two Encode calls.
func updateByteAvatar() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, map[slot.Position]uint32{}, map[int8]uint32{})
}

// TestMessengerUpdateByteOutput pins the MessengerUpdate (MESSENGER mode=7) wire
// body against the client read-order. The dispatcher CUIMessenger::OnPacket
// (@0x8511fc v83) reads Decode1(mode) then routes mode==7 to
// CUIMessenger::OnAvatar (@0x85194f), which reads:
//
//	Decode1(position)              [0x85196b]
//	AvatarLook::Decode(v6, a2)     [0x8519c1]   — single opaque DecodeBuf
//
// OnAvatar reads NO name and NO channelId (contrast MessengerAdd). Atlas's
// Update encoder emits mode(1) + position(1) + avatar-blob, matching exactly.
//
// AvatarLook is an OPAQUE_LEDGER VERIFIED-EXCEPTION (see
// docs/packets/audits/OPAQUE_LEDGER.md): the blob has no per-field decompile
// line — its bytes are derived from the independently-audited model.Avatar
// encoder. This test pins the non-opaque prefix exactly and asserts the trailing
// blob equals avatar.Encode(...) verbatim.
//
// packet-audit:verify packet=messenger/clientbound/MessengerUpdate version=gms_v83 ida=0x8511fc
// packet-audit:verify packet=messenger/clientbound/MessengerUpdate version=gms_v87 ida=0x8b978f
// packet-audit:verify packet=messenger/clientbound/MessengerUpdate version=gms_v95 ida=0x7f2ea0
// packet-audit:verify packet=messenger/clientbound/MessengerUpdate version=jms_v185 ida=0x8e4bab
// packet-audit:verify packet=messenger/clientbound/MessengerUpdate version=gms_v84 ida=0x87cbd8
func TestMessengerUpdateByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// Single-equipment avatar so the model.Avatar map iteration is
			// deterministic and the trailing opaque blob is byte-stable.
			ava := updateByteAvatar()
			in := NewMessengerUpdate(7, 1, ava)
			got := in.Encode(l, ctx)(nil)

			// Prefix: mode(1) + position(1).
			require.GreaterOrEqual(t, len(got), 2, "body has mode+position prefix")
			require.Equal(t, byte(7), got[0], "mode (dispatcher byte, Decode1 @0x851203)")
			require.Equal(t, byte(1), got[1], "position (Decode1 @0x85196b)")

			// Trailing opaque AvatarLook blob == model.Avatar encoder output
			// (OPAQUE_LEDGER VERIFIED-EXCEPTION; no per-field decompile line).
			wantBlob := ava.Encode(l, ctx)(nil)
			require.True(t, bytes.Equal(got[2:], wantBlob),
				"avatar blob must equal model.Avatar encoder output\n got=% x\nwant=% x", got[2:], wantBlob)

			// And nothing follows the avatar blob (no name/channelId trailer).
			require.Equal(t, 2+len(wantBlob), len(got), "no trailing name/channelId after avatar")
		})
	}
}
