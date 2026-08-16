package serverbound

import (
	"bytes"
	"testing"

	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// No packet-audit:verify markers here on purpose. This type decodes the SAME
// wire op as character/serverbound.CheckName (CHECK_CHAR_NAME) — the matrix
// cells for that op are owned by that codec's test, and claiming them a second
// time here would double-count a row without a fresh decompile. The
// CCashShop::SendCheckDuplicateIDPacket sender is not named in any checked-in
// IDB export; see this type's doc comment for exactly what the shape rests on.

func TestCheckNameChangeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CheckNameChangeRequest{name: "TestChar"}
			output := CheckNameChangeRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}

// TestCheckNameChangeByteOutput pins the wire on every variant: a single
// length-prefixed name and nothing else.
func TestCheckNameChangeByteOutput(t *testing.T) {
	want := []byte{
		0x08, 0x00, // EncodeStr length = 8
		0x54, 0x65, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, // "TestChar"
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := CheckNameChangeRequest{name: "TestChar"}.Encode(nil, ctx)(nil)
			if !bytes.Equal(got, want) {
				t.Errorf("CheckNameChangeRequest wire: got %x want %x", got, want)
			}
		})
	}
}

// TestCheckNameChangeMatchesLoginCheckName is the guard on the claim this
// codec's doc comment makes: the cash-shop rename probe and character
// creation's name check are the SAME serverbound op, so the two Go types must
// agree byte for byte on every variant. If someone later gates one of them on
// a version, this fails rather than letting the cash shop silently decode a
// login-shaped body.
func TestCheckNameChangeMatchesLoginCheckName(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cash := CheckNameChangeRequest{name: "TestChar"}.Encode(nil, ctx)(nil)
			login := charsb.NewCheckName("TestChar").Encode(nil, ctx)(nil)
			if !bytes.Equal(cash, login) {
				t.Errorf("cash-shop wire %x diverged from login wire %x", cash, login)
			}
		})
	}
}
