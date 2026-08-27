package clientbound

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// baseFrameHex is the fixed prefix every CharacterAppearanceUpdate frame
// shares regardless of version or ring content: characterId(4) + flags(1) +
// the fixture avatar block (AvatarLook::Decode's own byte layout is pinned
// independently by avatar_test.go, unchanged by this task). Transcribed once
// (verified byte-for-byte against a passing Encode() call) so the "empty" and
// "populated" cases below pin a known-good baseline rather than deriving it
// from the same production code path they are testing — the precedent set by
// spawn_test.go's "couple populated" case, which pins its whole-frame `empty`
// fixture as a hex literal and only derives the ring span under test.
const baseFrameHex = "7856341201010214000000011e000000ffff00000000000000000000000000000000"

// runAppearanceUpdateCases exercises the empty and populated ring cases for a
// given tenant context. Only the frame shape this task changes — the ring
// block and the version-gated trailing completed-set int — is derived from
// production codec calls; the rest of the frame is the pinned baseFrameHex.
//
// The ring block itself is built via model.RingSet.EncodeField, matching the
// precedent already established by spawn_test.go's "couple populated" case:
// this proves CharacterAppearanceUpdate wires the shared ring codec (Task
// 2/3, tested in its own right in model/ring_test.go), not that the codec's
// byte layout is correct in isolation.
func runAppearanceUpdateCases(t *testing.T, ctx context.Context) {
	t.Helper()
	tn := tenant.MustFromContext(ctx)

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

	base, err := hex.DecodeString(baseFrameHex)
	if err != nil {
		t.Fatalf("baseFrameHex: %v", err)
	}

	// hasTrailing mirrors the production gate (hasTrailingCompletedSetItemId
	// in appearance_update.go): true only for gms_v87/gms_v95, where the
	// client reads nCompletedSetItemID unconditionally after the marriage
	// arm (gms_v87.json @0xa090f4, trailing Decode4 @0xa092d5; gms_v95.json
	// @0x954110, trailing Decode4 @0x9542ec). False for gms_v83, gms_v84,
	// gms_v79, and jms_v185, none of which read a trailing int there.
	hasTrailing := hasTrailingCompletedSetItemId(tn)

	t.Run("empty", func(t *testing.T) {
		rings := model.RingSet{}
		input := NewCharacterAppearanceUpdate(0x12345678, avatar, rings)
		got := input.Encode(nil, ctx)(nil)

		w := response.NewWriter(nil)
		w.WriteByteArray(base)
		rings.EncodeField(w, tn)
		if hasTrailing {
			w.WriteInt(0) // nCompletedSetItemID (gms_v87/v95 only)
		}
		want := w.Bytes()

		if !bytes.Equal(got, want) {
			t.Errorf("empty-ring bytes:\n got %x\nwant %x", got, want)
		}

		// The frame correction: the pre-task encoder wrote an unconditional
		// trailing WriteInt(0) "completed set item id" on every version. That
		// is correct only for gms_v87/gms_v95 (see hasTrailing above); every
		// other version now omits it (design.md §3.2's "one int long" defect,
		// scoped to the versions the client actually doesn't read it on).
		// Assert the delta explicitly — computed independently of `got` —
		// rather than leaving it implied by the byte-equality check above:
		// base frame + 3 empty ring flags + the trailing int(4), included
		// only when hasTrailing.
		wantLen := len(base) + 3
		if hasTrailing {
			wantLen += 4
		}
		if len(got) != wantLen {
			t.Fatalf("empty-ring length: got %d want %d", len(got), wantLen)
		}
	})

	t.Run("populated", func(t *testing.T) {
		// fixture shared with Task 2 (model/ring_test.go) and Task 5's
		// spawn-wiring test (spawn_test.go "couple populated" case).
		fixturePartnerSNU := uint64(0x99AABBCCDDEEFF00)
		fixturePartnerSN := int64(fixturePartnerSNU)
		couple := &model.PairRing{OwnSN: 0x1122334455667788, PartnerSN: fixturePartnerSN, ItemId: 0x00001234}
		rings := model.RingSet{Couple: couple}

		input := NewCharacterAppearanceUpdate(0x12345678, avatar, rings)
		got := input.Encode(nil, ctx)(nil)

		w := response.NewWriter(nil)
		w.WriteByteArray(base)
		rings.EncodeField(w, tn)
		if hasTrailing {
			w.WriteInt(0) // nCompletedSetItemID (gms_v87/v95 only)
		}
		want := w.Bytes()

		if !bytes.Equal(got, want) {
			t.Errorf("populated-ring bytes:\n got %x\nwant %x", got, want)
		}
	})
}

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
// Atlas always writes flags=1 (look-only), so only the &1 branch fires. The
// marriage if/else has no trailing unconditional Decode4 at v83 (the else
// branch zeroes the slots and reads nothing) — the pre-task encoder's
// trailing WriteInt(0) "completed set item id" was benign, unread slack here
// (design.md §3.2's "one int long" defect); the encoder no longer writes one
// at v83.
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
	runAppearanceUpdateCases(t, ctx)
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
// Atlas always writes flags=1 (look-only); the marriage if/else has no
// trailing unconditional Decode4 at v84, as at v83.
func TestCharacterAppearanceUpdateByteOutputV84(t *testing.T) {
	v84 := pt.Variants[5] // GMS v84
	ctx := pt.CreateContext(v84.Region, v84.MajorVersion, v84.MinorVersion)
	runAppearanceUpdateCases(t, ctx)
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
// Unlike v83/v84 (trailing int is unread slack when marriage==0), v87 reads
// the trailing Decode4 (completed-set-item id) unconditionally — it sits
// AFTER the marriage if/else, not inside it. Atlas writes flags=1 (look-only)
// and the ring block per RingSet, then writes the trailing WriteInt(0) to
// match this unconditional read (hasTrailingCompletedSetItemId gate:
// IsRegion("GMS") && MajorAtLeast(87)).
//
// AvatarLook::Decode (v87 @0x508277):
//
//	gender=Decode1, skin=Decode1, face=Decode4, !mega=Decode1, hair=Decode4,
//	equip loop (key Decode1; 0xFF terminates; else value Decode4),
//	masked loop (same), cashWeapon=Decode4, pets=DecodeBuffer(12)=3xDecode4.
func TestCharacterAppearanceUpdateByteOutputV87(t *testing.T) {
	v87 := pt.Variants[2] // GMS v87
	ctx := pt.CreateContext(v87.Region, v87.MajorVersion, v87.MinorVersion)
	runAppearanceUpdateCases(t, ctx)
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
// The read order is byte-identical to v87: flags, AvatarLook, three ring
// markers, then the trailing completed-set int consumed unconditionally
// (@0x9542ec), same as at v87 (@0xa092d5). Atlas writes the matching trailing
// WriteInt(0) here too (hasTrailingCompletedSetItemId gate: IsRegion("GMS")
// && MajorAtLeast(87)).
//
// AvatarLook::Decode (v95 @0x4f2c00):
//
//	gender=Decode1, skin=Decode1, face=Decode4, !mega=Decode1, hair=Decode4,
//	equip loop (key Decode1; 0xFF terminates; else value Decode4),
//	masked loop (same), cashWeapon=Decode4, pets=DecodeBuffer(12)=3xDecode4.
func TestCharacterAppearanceUpdateByteOutputV95(t *testing.T) {
	v95 := pt.Variants[3] // GMS v95
	ctx := pt.CreateContext(v95.Region, v95.MajorVersion, v95.MinorVersion)
	runAppearanceUpdateCases(t, ctx)
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
// As in v83/v84, the marriage if/else has no trailing unconditional Decode4 in
// jms. Atlas writes flags=1 (look-only); the ring block wire for JMS carries
// the extra per-arm entry-count int (model.RingSet.EncodeField, isJMS branch),
// which runAppearanceUpdateCases builds via the production codec rather than
// re-transcribing here.
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
	runAppearanceUpdateCases(t, ctx)
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
// As in v83/v84/jms, the marriage if/else has no trailing unconditional
// Decode4 in v79. Atlas writes flags=1 (look-only) and the ring block per
// RingSet.
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
	runAppearanceUpdateCases(t, ctx)
}

func TestCharacterAppearanceUpdateRoundTrip(t *testing.T) {
	v83 := pt.Variants[1]
	ctx := pt.CreateContext(v83.Region, v83.MajorVersion, v83.MinorVersion)
	avatar := model.NewAvatar(1, 2, 0x14, false, 0x1E, nil, nil, nil)
	input := NewCharacterAppearanceUpdate(0x12345678, avatar, model.RingSet{})
	output := CharacterAppearanceUpdate{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v want %v", output.CharacterId(), input.CharacterId())
	}
}
