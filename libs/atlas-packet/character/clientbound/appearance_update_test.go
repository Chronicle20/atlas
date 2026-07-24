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
//
//	gender=Decode1, skin=Decode1, face=Decode4, !mega=Decode1, hair=Decode4,
//	equip loop (key Decode1; 0xFF terminates; else value Decode4),
//	masked loop (same), cashWeapon=Decode4, pets=DecodeBuffer(12)=3xDecode4.
//
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v83 ida=0x98367e
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v84 ida=0x9c3a1c
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v87 ida=0xa090f4
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v95 ida=0x954110
func TestCharacterAppearanceUpdateByteOutput(t *testing.T) {
	v83 := pt.Variants[1] // GMS v83
	ctx := pt.CreateContext(v83.Region, v83.MajorVersion, v83.MinorVersion)

	// Empty equipment/masked maps + nil pets keep the avatar block deterministic
	// (the encoder ranges over maps, whose iteration order is unstable otherwise).
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
		0x78, 0x56, 0x34, 0x12, // characterId (WriteInt)                 /*0x983697 dispatch*/
		0x01, // flags = 1 (look-only)                  /*0x983697*/
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
		0x01, // flags = 1 (look-only)                  /*0x9c3a35*/
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
//
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
		0x01, // flags = 1 (look-only)                  /*0xa0910d*/
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

// CharacterAppearanceUpdate v95 byte-fixture.
//
// Client read order — CUserRemote::OnAvatarModified (v95 @0x954110):
//
//	flags = Decode1                  // bitfield: &1 look, &2 speed, &4 carry  /*0x954122*/
//	if flags & 1:  AvatarLook::Decode  // full avatar/look block (@0x4f2c00)   /*0x954131*/
//	if flags & 2:  Decode1            // riding/vehicle speed (not set here)     /*0x954185*/
//	if flags & 4:  Decode1            // carry-item effect (not set here)        /*0x9541a5*/
//	crushMarker    = Decode1         // 0 => no buffer; else 2xDecodeBuffer(8)+Decode4 /*0x9541b7*/
//	friendMarker   = Decode1         // 0 => no buffer; else 2xDecodeBuffer(8)+Decode4 /*0x954202*/
//	marriageMarker = Decode1         // != 0 => 3x Decode4; else zeros          /*0x954251*/
//	completedSet   = Decode4         // read UNCONDITIONALLY in v95 (m_nCompletedSetItemID) /*0x9542ec*/
//
// The read order is byte-identical to v83/v84/v87: flags, AvatarLook, three ring
// markers, then the trailing completed-set int (consumed unconditionally at v95 as
// at v87, @0x9542ec). Atlas writes flags=1 (look-only) and the three ring markers
// as 0, so the wire bytes are unchanged from the v87 fixture.
//
// AvatarLook::Decode (v95 @0x4f2c00):
//
//	gender=Decode1, skin=Decode1, face=Decode4, !mega=Decode1, hair=Decode4,
//	equip loop (key Decode1; 0xFF terminates; else value Decode4),
//	masked loop (same), cashWeapon=Decode4, pets=DecodeBuffer(12)=3xDecode4.
func TestCharacterAppearanceUpdateByteOutputV95(t *testing.T) {
	v95 := pt.Variants[3] // GMS v95
	ctx := pt.CreateContext(v95.Region, v95.MajorVersion, v95.MinorVersion)

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
		0x78, 0x56, 0x34, 0x12, // characterId (WriteInt)                 /*0x954110 dispatch*/
		0x01, // flags = 1 (look-only)                  /*0x954122*/
		// --- AvatarLook block (flags & 1) ---                            /*0x954131*/
		0x01,                   // gender (Decode1)                       /*0x4f2c13*/
		0x02,                   // skinColor (Decode1)                    /*0x4f2c20*/
		0x14, 0x00, 0x00, 0x00, // face (Decode4)                         /*0x4f2c33*/
		0x01,                   // !mega -> WriteBool(true) (Decode1)     /*0x4f2c53*/
		0x1e, 0x00, 0x00, 0x00, // hair (Decode4)                         /*0x4f2c61*/
		0xff,                   // equipment terminator                   /*0x4f2c6d*/
		0xff,                   // masked-equipment terminator            /*0x4f2cb3*/
		0x00, 0x00, 0x00, 0x00, // cash weapon (Decode4)                  /*0x4f2cf6*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12 = 3x Decode4)   /*0x4f2d04*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2
		// --- ring markers ---
		0x00,                   // crush ring marker (Decode1)            /*0x9541b7*/
		0x00,                   // friendship ring marker (Decode1)       /*0x954202*/
		0x00,                   // marriage ring marker (Decode1)         /*0x954251*/
		0x00, 0x00, 0x00, 0x00, // completed set item id (Decode4, read unconditionally) /*0x9542ec*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("appearance-update v95 bytes:\n got %x\nwant %x", got, want)
	}
}

// CharacterAppearanceUpdate jms byte-fixture.
//
// Client read order — CUserRemote::OnAvatarModified (jms v185 @0xa57221,
// MapleStory_dump_SCY.exe):
//
//	flags = Decode1                  // bitfield: &1 look, &2 speed, &4 carry  /*0xa57230*/
//	if flags & 1:  AvatarLook::Decode  // full avatar/look block (@0x51517e)   /*0xa57246*/
//	if flags & 2:  Decode1            // riding/vehicle speed (not set here)     /*0xa57295*/
//	if flags & 4:  Decode1            // carry-item effect (not set here)        /*0xa572b7*/
//	crushMarker    = Decode1         // 0 => no buffer; else Decode4 count loop  /*0xa572ca*/
//	friendMarker   = Decode1         // 0 => no buffer; else Decode4 count loop  /*0xa5733b*/
//	marriageMarker = Decode1         // != 0 => 3x Decode4; else zeros           /*0xa573af*/
//
// As in v83/v84, the marriage if/else has NO trailing unconditional Decode4 in jms
// (the else branch @0xa573e1 zeroes the slots and reads nothing). Atlas always writes
// flags=1 (look-only) and the three ring markers as 0, so the trailing WriteInt(0)
// "completed set item id" is benign trailing slack (unread when marriage==0). The
// jms wire matches the GMS-shaped codec — no per-version delta.
//
// AvatarLook::Decode (jms @0x51517e):
//
//	gender=Decode1 /*0x51518a*/, skin=Decode1 /*0x515194*/, face=Decode4 /*0x5151a1*/,
//	!mega=Decode1 /*0x5151ce*/, hair=Decode4 /*0x5151d5*/, equip loop (key Decode1
//	0xFF term; value Decode4), masked loop (same), cashWeapon=Decode4 /*0x515251*/,
//	pets=DecodeBuffer(12) /*0x515264*/.
//
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=jms_v185 ida=0xa57221
func TestCharacterAppearanceUpdateByteOutputJMS(t *testing.T) {
	jms := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(jms.Region, jms.MajorVersion, jms.MinorVersion)

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
		0x78, 0x56, 0x34, 0x12, // characterId (WriteInt)                 /*0xa57221 dispatch*/
		0x01, // flags = 1 (look-only)                  /*0xa57230*/
		// --- AvatarLook block (flags & 1) ---                            /*0xa57246*/
		0x01,                   // gender (Decode1)                       /*0x51518a*/
		0x02,                   // skinColor (Decode1)                    /*0x515194*/
		0x14, 0x00, 0x00, 0x00, // face (Decode4)                         /*0x5151a1*/
		0x01,                   // !mega -> WriteBool(true) (Decode1)     /*0x5151ce*/
		0x1e, 0x00, 0x00, 0x00, // hair (Decode4)                         /*0x5151d5*/
		0xff,                   // equipment terminator                   /*0x5151de*/
		0xff,                   // masked-equipment terminator            /*0x515215*/
		0x00, 0x00, 0x00, 0x00, // cash weapon (Decode4)                  /*0x515251*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12 = 3x Decode4)   /*0x515264*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2
		// --- ring markers ---
		0x00,                   // crush ring marker (Decode1)            /*0xa572ca*/
		0x00,                   // friendship ring marker (Decode1)       /*0xa5733b*/
		0x00,                   // marriage ring marker (Decode1)         /*0xa573af*/
		0x00, 0x00, 0x00, 0x00, // completed set item id (trailing slack; unread when marriage==0)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("appearance-update jms bytes:\n got %x\nwant %x", got, want)
	}
}

// CharacterAppearanceUpdate v79 byte-fixture.
//
// Client read order — CUserRemote::OnAvatarModified (GMS_v79_1_DEVM.exe @0x8d9824):
//
//	flags = Decode1                  // bitfield: &1 look, &2 speed, &4 carry  /*0x8d983d*/
//	if flags & 1:  AvatarLook::Decode  // full avatar/look block (@0x4db6dd)    /*0x8d9849*/
//	if flags & 2:  Decode1            // riding/vehicle speed (not set here)     /*0x8d989d*/
//	if flags & 4:  Decode1            // carry-item effect (not set here)        /*0x8d98bf*/
//	crushMarker    = Decode1         // 0 => no buffer; else 2xDecodeBuffer(8)+Decode4 /*0x8d98d1*/
//	friendMarker   = Decode1         // 0 => no buffer; else 2xDecodeBuffer(8)+Decode4 /*0x8d991e*/
//	marriageMarker = Decode1         // != 0 => 3x Decode4; else zeros          /*0x8d996b*/
//
// As in v83/v84/jms, the marriage if/else has NO trailing unconditional Decode4 in v79
// (the else branch @0x8d99b0 zeroes the slots and reads nothing). Atlas always writes
// flags=1 (look-only) and the three ring markers as 0, so the trailing WriteInt(0)
// "completed set item id" is benign trailing slack. The wire is byte-identical to v83.
//
// AvatarLook::Decode (v79 @0x4db6dd):
//
//	gender=Decode1 /*0x4db6f0*/, skin=Decode1 /*0x4db6fd*/, face=Decode4 /*0x4db711*/,
//	!mega=Decode1 /*0x4db72d*/, hair=Decode4 /*0x4db739*/, equip loop (key Decode1
//	0xFF term; value Decode4), masked loop (same), cashWeapon=Decode4 /*0x4db7b5*/,
//	pets=DecodeBuffer(12)=3xDecode4 /*0x4db7c8*/.
//
// packet-audit:verify packet=character/clientbound/CharacterAppearanceUpdate version=gms_v79 ida=0x8d9824
func TestCharacterAppearanceUpdateByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)

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
		0x78, 0x56, 0x34, 0x12, // characterId (WriteInt)                 /*0x8d9824 dispatch*/
		0x01, // flags = 1 (look-only)                  /*0x8d983d*/
		// --- AvatarLook block (flags & 1) ---                            /*0x8d9849*/
		0x01,                   // gender (Decode1)                       /*0x4db6f0*/
		0x02,                   // skinColor (Decode1)                    /*0x4db6fd*/
		0x14, 0x00, 0x00, 0x00, // face (Decode4)                         /*0x4db711*/
		0x01,                   // !mega -> WriteBool(true) (Decode1)     /*0x4db72d*/
		0x1e, 0x00, 0x00, 0x00, // hair (Decode4)                         /*0x4db739*/
		0xff,                   // equipment terminator                   /*0x4db742*/
		0xff,                   // masked-equipment terminator            /*0x4db779*/
		0x00, 0x00, 0x00, 0x00, // cash weapon (Decode4)                  /*0x4db7b5*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12 = 3x Decode4)   /*0x4db7c8*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2
		// --- ring markers ---
		0x00,                   // crush ring marker (Decode1)            /*0x8d98d1*/
		0x00,                   // friendship ring marker (Decode1)       /*0x8d991e*/
		0x00,                   // marriage ring marker (Decode1)         /*0x8d996b*/
		0x00, 0x00, 0x00, 0x00, // completed set item id (trailing slack; unread when marriage==0)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("appearance-update v79 bytes:\n got %x\nwant %x", got, want)
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
