package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CharacterAppearanceUpdate byte-fixture.
//
// Client read order — CUserRemote::OnAvatarModified (v83 @0x98367e):
//
//	flags = Decode1                  // bitfield: &1 look, &2 speed, &4 carry  /*0x983697*/
//	if flags & 1:  AvatarLook::Decode  // full avatar/look block               /*0x9836a3*/
//	crushMarker    = Decode1         // 0 => no buffer; else DecodeBuffer(16)+Decode4 /*0x98372b*/
//	friendMarker   = Decode1         // 0 => no buffer; else DecodeBuffer(16)+Decode4 /*0x983778*/
//	marriageMarker = Decode1         // != 0 => 3x Decode4; else zeros          /*0x9837c5*/
//
// Atlas always writes flags=1 (look-only), so only the &1 branch fires. The three
// ring markers are written as 0 (no following buffers). The trailing WriteInt(0)
// "completed set item id" is NOT read by the client when marriageMarker==0 (the
// else branch zeroes the slots and reads nothing) — it is benign trailing slack,
// the existing codec shape, version-independent.
//
// AvatarLook::Decode (v83 @0x4e749a):
//	gender=Decode1, skin=Decode1, face=Decode4, !mega=Decode1, hair=Decode4,
//	equip loop (key Decode1; 0xFF terminates; else value Decode4),
//	masked loop (same), cashWeapon=Decode4, pets=DecodeBuffer(12)=3xDecode4.
//
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v83 ida=0x98367e
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v84 ida=0x9c3a1c
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v87 ida=0xa090f4
func TestCharacterAppearanceUpdateByteOutput(t *testing.T) {
	v83 := pt.Variants[1] // GMS v83
	ctx := pt.CreateContext(v83.Region, v83.MajorVersion, v83.MinorVersion)

	// Empty equipment/masked maps + nil pets keep the avatar block deterministic
	// (the encoder ranges over maps, whose iteration order is unstable otherwise).
	avatar := model.NewAvatar(
		1,      // gender
		2,      // skinColor
		0x14,   // face
		false,  // mega -> WriteBool(!mega)=WriteBool(true)=0x01
		0x1E,   // hair
		nil,    // equipment (-> just 0xFF terminator)
		nil,    // maskedEquipment (-> just 0xFF terminator)
		nil,    // pets (-> 3x WriteInt(0))
	)
	input := NewCharacterAppearanceUpdate(0x12345678, avatar)
	got := input.Encode(nil, ctx)(nil)

	want := []byte{
		0x78, 0x56, 0x34, 0x12, // characterId (WriteInt)                 /*0x983697 dispatch*/
		0x01,                   // flags = 1 (look-only)                  /*0x983697*/
		// --- AvatarLook block (flags & 1) ---                            /*0x9836a3*/
		0x01,                   // gender (Decode1)                       /*0x4e74ad*/
		0x02,                   // skinColor (Decode1)                    /*0x4e74ba*/
		0x14, 0x00, 0x00, 0x00, // face (Decode4)                         /*0x4e74ce*/
		0x01,                   // !mega -> WriteBool(true) (Decode1)     /*0x4e74ea*/
		0x1e, 0x00, 0x00, 0x00, // hair (Decode4)                         /*0x4e74f6*/
		0xff,                   // equipment terminator                   /*0x4e74ff*/
		0xff,                   // masked-equipment terminator            /*0x4e7536*/
		0x00, 0x00, 0x00, 0x00, // cash weapon (Decode4)                  /*0x4e7572*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12 = 3x Decode4)   /*0x4e7585*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2
		// --- ring markers ---
		0x00,                   // crush ring marker (Decode1)            /*0x98372b*/
		0x00,                   // friendship ring marker (Decode1)       /*0x983778*/
		0x00,                   // marriage ring marker (Decode1)         /*0x9837c5*/
		0x00, 0x00, 0x00, 0x00, // completed set item id (trailing slack; unread when marriage==0)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("appearance-update bytes:\n got %x\nwant %x", got, want)
	}
}

// CharacterAppearanceUpdate v84 byte-fixture.
//
// Client read order — CUserRemote::OnAvatarModified (v84 @0x9c3a1c), byte-identical
// to v83 (v84 body ≡ v83 below ~0x3D, IDA-confirmed):
//
//	flags = Decode1                  // bitfield: &1 look, &2 speed, &4 carry  /*0x9c3a35*/
//	if flags & 1:  AvatarLook::Decode  // full avatar/look block (@0x4ef958)   /*0x9c3a41*/
//	crushMarker    = Decode1         // 0 => no buffer; else 2xDecodeBuffer(8)+Decode4 /*0x9c3ac9*/
//	friendMarker   = Decode1         // 0 => no buffer; else 2xDecodeBuffer(8)+Decode4 /*0x9c3b16*/
//	marriageMarker = Decode1         // != 0 => 3x Decode4; else zeros          /*0x9c3b63*/
//
// Atlas always writes flags=1 (look-only); the three ring markers are written as 0.
// The trailing WriteInt(0) is unread by the client when marriageMarker==0 (the else
// branch @0x9c3ba8 zeroes the slots and reads nothing) — benign trailing slack.
//
// The wire bytes mirror the v83 fixture exactly.
func TestCharacterAppearanceUpdateByteOutputV84(t *testing.T) {
	v84 := pt.Variants[5] // GMS v84
	ctx := pt.CreateContext(v84.Region, v84.MajorVersion, v84.MinorVersion)

	avatar := model.NewAvatar(
		1,     // gender
		2,     // skinColor
		0x14,  // face
		false, // mega -> WriteBool(!mega)=WriteBool(true)=0x01
		0x1E,  // hair
		nil,   // equipment (-> just 0xFF terminator)
		nil,   // maskedEquipment (-> just 0xFF terminator)
		nil,   // pets (-> 3x WriteInt(0))
	)
	input := NewCharacterAppearanceUpdate(0x12345678, avatar)
	got := input.Encode(nil, ctx)(nil)

	want := []byte{
		0x78, 0x56, 0x34, 0x12, // characterId (WriteInt)                 /*0x9c3a1c dispatch*/
		0x01,                   // flags = 1 (look-only)                  /*0x9c3a35*/
		// --- AvatarLook block (flags & 1) ---                            /*0x9c3a41*/
		0x01,                   // gender (Decode1)                       /*0x4ef96b*/
		0x02,                   // skinColor (Decode1)                    /*0x4ef978*/
		0x14, 0x00, 0x00, 0x00, // face (Decode4)                         /*0x4ef98c*/
		0x01,                   // !mega -> WriteBool(true) (Decode1)     /*0x4ef9a8*/
		0x1e, 0x00, 0x00, 0x00, // hair (Decode4)                         /*0x4ef9b4*/
		0xff,                   // equipment terminator                   /*0x4ef9bd*/
		0xff,                   // masked-equipment terminator            /*0x4ef9f4*/
		0x00, 0x00, 0x00, 0x00, // cash weapon (Decode4)                  /*0x4efa30*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12 = 3x Decode4)   /*0x4efa43*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2
		// --- ring markers ---
		0x00,                   // crush ring marker (Decode1)            /*0x9c3ac9*/
		0x00,                   // friendship ring marker (Decode1)       /*0x9c3b16*/
		0x00,                   // marriage ring marker (Decode1)         /*0x9c3b63*/
		0x00, 0x00, 0x00, 0x00, // completed set item id (trailing slack; unread when marriage==0)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("appearance-update v84 bytes:\n got %x\nwant %x", got, want)
	}
}

// CharacterAppearanceUpdate v87 byte-fixture.
//
// Client read order — CUserRemote::OnAvatarModified (v87 @0xa090f4):
//
//	flags = Decode1                  // bitfield: &1 look, &2 speed, &4 carry  /*0xa0910d*/
//	if flags & 1:  AvatarLook::Decode  // full avatar/look block (@0x508277)   /*0xa09119*/
//	if flags & 2:  Decode1            // riding/vehicle speed (not set here)     /*0xa0916d*/
//	if flags & 4:  Decode1            // carry-item effect (not set here)        /*0xa0918f*/
//	crushMarker    = Decode1         // 0 => no buffer; else 2xDecodeBuffer(8)+Decode4 /*0xa091a1*/
//	friendMarker   = Decode1         // 0 => no buffer; else 2xDecodeBuffer(8)+Decode4 /*0xa091ee*/
//	marriageMarker = Decode1         // != 0 => 3x Decode4; else zeros          /*0xa0923b*/
//	completedSet   = Decode4         // read UNCONDITIONALLY in v87 (this[1456]) /*0xa092d5*/
//
// Unlike the v83/v84 comment ("trailing int unread when marriage==0"), v87 reads
// the trailing Decode4 (completed-set-item id) unconditionally — it sits AFTER the
// marriage if/else, not inside it. Atlas writes flags=1 (look-only) and the three
// ring markers as 0, so the wire bytes are still byte-identical to v83/v84; only
// the client's read-disposition of the final int differs (now consumed, not slack).
//
// AvatarLook::Decode (v87 @0x508277):
//	gender=Decode1, skin=Decode1, face=Decode4, !mega=Decode1, hair=Decode4,
//	equip loop (key Decode1; 0xFF terminates; else value Decode4),
//	masked loop (same), cashWeapon=Decode4, pets=DecodeBuffer(12)=3xDecode4.
func TestCharacterAppearanceUpdateByteOutputV87(t *testing.T) {
	v87 := pt.Variants[2] // GMS v87
	ctx := pt.CreateContext(v87.Region, v87.MajorVersion, v87.MinorVersion)

	avatar := model.NewAvatar(
		1,     // gender
		2,     // skinColor
		0x14,  // face
		false, // mega -> WriteBool(!mega)=WriteBool(true)=0x01
		0x1E,  // hair
		nil,   // equipment (-> just 0xFF terminator)
		nil,   // maskedEquipment (-> just 0xFF terminator)
		nil,   // pets (-> 3x WriteInt(0))
	)
	input := NewCharacterAppearanceUpdate(0x12345678, avatar)
	got := input.Encode(nil, ctx)(nil)

	want := []byte{
		0x78, 0x56, 0x34, 0x12, // characterId (WriteInt)                 /*0xa090f4 dispatch*/
		0x01,                   // flags = 1 (look-only)                  /*0xa0910d*/
		// --- AvatarLook block (flags & 1) ---                            /*0xa09119*/
		0x01,                   // gender (Decode1)                       /*0x50828a*/
		0x02,                   // skinColor (Decode1)                    /*0x508297*/
		0x14, 0x00, 0x00, 0x00, // face (Decode4)                         /*0x5082ab*/
		0x01,                   // !mega -> WriteBool(true) (Decode1)     /*0x5082c7*/
		0x1e, 0x00, 0x00, 0x00, // hair (Decode4)                         /*0x5082d3*/
		0xff,                   // equipment terminator                   /*0x5082dc*/
		0xff,                   // masked-equipment terminator            /*0x508313*/
		0x00, 0x00, 0x00, 0x00, // cash weapon (Decode4)                  /*0x50834f*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12 = 3x Decode4)   /*0x50835d*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2
		// --- ring markers ---
		0x00,                   // crush ring marker (Decode1)            /*0xa091a1*/
		0x00,                   // friendship ring marker (Decode1)       /*0xa091ee*/
		0x00,                   // marriage ring marker (Decode1)         /*0xa0923b*/
		0x00, 0x00, 0x00, 0x00, // completed set item id (Decode4, read unconditionally) /*0xa092d5*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("appearance-update v87 bytes:\n got %x\nwant %x", got, want)
	}
}

func TestCharacterAppearanceUpdateRoundTrip(t *testing.T) {
	v83 := pt.Variants[1]
	ctx := pt.CreateContext(v83.Region, v83.MajorVersion, v83.MinorVersion)
	avatar := model.NewAvatar(1, 2, 0x14, false, 0x1E, nil, nil, nil)
	input := NewCharacterAppearanceUpdate(0x12345678, avatar)
	output := CharacterAppearanceUpdate{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v want %v", output.CharacterId(), input.CharacterId())
	}
}
